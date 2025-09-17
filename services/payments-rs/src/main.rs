// services/payments-rs/src/main.rs

use tonic::{transport::Server, Request, Response, Status, metadata::MetadataMap};
use tracing::info;
use uuid::Uuid;

mod generated {
    pub mod common {
        pub mod v1 { tonic::include_proto!("common.v1"); }
    }
    pub mod db {
        pub mod v1 { tonic::include_proto!("db.v1"); }
    }
    pub mod payments {
        pub mod v1 { tonic::include_proto!("payments.v1"); }
    }
}

// alias modul agar singkat
use generated::db::v1 as dbv1;
use generated::payments::v1 as payv1;

// client & server traits
use dbv1::db_client::DbClient;
use payv1::payments_service_server::{PaymentsService, PaymentsServiceServer};

// tipe-tipe yang sering dipakai
use dbv1::{ReserveFundsRequest, CommitReservationRequest, RollbackReservationRequest};
use payv1::{CreatePaymentRequest, CreatePaymentResponse, CreatePaymentRequest, LogAndSettleResponse};

mod idempo;
use idempo::IdempoCache;

// ------ helper ambil idempotency-key ------
fn get_idempo_from_meta(meta: &MetadataMap) -> Option<String> {
    meta.get("idempotency-key")
        .and_then(|v| v.to_str().ok())
        .map(|s| s.to_string())
}

// ------ service impl ------
#[derive(Clone)]
struct PaymentsSvc {
    db_addr: String,
    cache: IdempoCache,
}

#[tonic::async_trait]
impl PaymentsService for PaymentsSvc {
    async fn create_payment(
        &self,
        _req: Request<CreatePaymentRequest>,
    ) -> Result<Response<CreatePaymentResponse>, Status> {
        // Stub sederhana agar trait terpenuhi; sesuaikan kalau mau dipakai beneran
        Ok(Response::new(CreatePaymentResponse {
            payment_id: Uuid::new_v4().to_string(),
            status: "PENDING".into(),
        }))
    }

    async fn log_and_settle(
        &self,
        req: Request<CreatePaymentRequest>,
    ) -> Result<Response<LogAndSettleResponse>, Status> {
        // 1) idempotency-key
        let meta_key = get_idempo_from_meta(req.metadata());
        let mut r = req.into_inner();
       
        let key = if !r.idempotency_key.is_empty() {
    r.idempotency_key.clone()
} else if let Some(m) = meta_key {
    m
} else {
    Uuid::new_v4().to_string()
};
r.idempotency_key = key.clone(); // karena field bertipe String


        // 2) fast-replay cache
        if let Some(hit) = self.cache.get(&key).await {
            return Ok(Response::new(LogAndSettleResponse {
                status: "SUCCESS_REPLAY".into(),
                message: hit,
                reservation_id: "".into(),
            }));
        }

        // 3) call db-rs: Reserve
        let mut db = DbClient::connect(format!("http://{}", self.db_addr))
            .await
            .map_err(|e| Status::unavailable(e.to_string()))?;

        let reserve = db
            .reserve_funds(ReserveFundsRequest {
                idempotency_key: key.clone(),
                sender_id: r.sender_id.clone(),
                receiver_id: r.receiver_id.clone(),
                amount_idr: r.amount_idr,
                currency_input: r.currency.clone(),
                created_at_iso: r.tx_date.clone(),
            })
            .await
            .map_err(|e| Status::internal(e.to_string()))?
            .into_inner();

        // Aman dari perbedaan nama enum: gunakan from_i32 + as_str_name()
        let res_status_name = dbv1::TxStatus::from_i32(reserve.status)
            .map(|s| s.as_str_name().to_string())
            .unwrap_or_default();

        if res_status_name == "TX_STATUS_INSUFFICIENT" {
            let res = LogAndSettleResponse {
                status: "FAILED".into(),
                message: "insufficient_funds".into(),
                reservation_id: "".into(),
            };
            self.cache.put(key.clone(), res.message.clone()).await;
            return Ok(Response::new(res));
        } else if res_status_name == "TX_STATUS_DUPLICATE" {
            let res = LogAndSettleResponse {
                status: "SUCCESS_REPLAY".into(),
                message: "duplicate_idempotency_key".into(),
                reservation_id: "".into(),
            };
            self.cache.put(key.clone(), res.message.clone()).await;
            return Ok(Response::new(res));
        } else if res_status_name != "TX_STATUS_OK" {
            return Err(Status::internal("reserve_error"));
        }

        // 4) Commit
        let commit = db
            .commit_reservation(CommitReservationRequest {
                reservation_id: reserve.reservation_id.clone(),
                idempotency_key: key.clone(),
            })
            .await
            .map_err(|e| Status::internal(e.to_string()))?
            .into_inner();

        let commit_status_name = dbv1::TxStatus::from_i32(commit.status)
            .map(|s| s.as_str_name().to_string())
            .unwrap_or_default();

        if commit_status_name != "TX_STATUS_OK" {
            // rollback untuk safety (best-effort)
            let _ = db
                .rollback_reservation(RollbackReservationRequest {
                    reservation_id: reserve.reservation_id.clone(),
                    reason: "commit_failed".into(),
                })
                .await;
            return Err(Status::internal("commit_failed"));
        }

        let res = LogAndSettleResponse {
            status: "SUCCESS".into(),
            message: "committed".into(),
            reservation_id: reserve.reservation_id,
        };
        self.cache.put(key, res.message.clone()).await;
        Ok(Response::new(res))
    }
}

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt().with_env_filter("info").init();

    let db_addr = std::env::var("DB_ADDR").unwrap_or_else(|_| "127.0.0.1:19095".into());
    let addr = "0.0.0.0:9096".parse().unwrap();

    let svc = PaymentsSvc {
        db_addr,
        cache: IdempoCache::new(),
    };

    // metrics sederhana di :9106
    tokio::spawn(async move {
    use axum::{routing::get, Router};
    use prometheus::{Encoder, Registry, TextEncoder};

    let registry = Registry::new();
    // TODO: register metrics ke `registry` kalau ada metrik custom

    let app = Router::new().route("/", get({
        let registry = registry.clone();
        move || async move {
            let mut buf = Vec::new();
            let enc = TextEncoder::new();
            enc.encode(&registry.gather(), &mut buf).unwrap();
            String::from_utf8(buf).unwrap()
        }
    }));

    let listener = tokio::net::TcpListener::bind("0.0.0.0:9106").await.unwrap();
    axum::serve(listener, app).await.unwrap();
});

    info!("payments-rs gRPC on {}", addr);
    Server::builder()
        .add_service(PaymentsServiceServer::new(svc))
        .serve(addr)
        .await?;

    Ok(())
}
