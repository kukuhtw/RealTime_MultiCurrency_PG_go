// services/payments-rs/src/main.rs

use tonic::{transport::Server, Request, Response, Status, metadata::MetadataMap, service::Interceptor};
use tracing::{info, warn};
use uuid::Uuid;

pub mod dbv1 { tonic::include_proto!("db.v1"); }
use dbv1::db_client::DbClient;
use dbv1::*;

pub mod paymentsv1 {
    tonic::include_proto!("payments.v1");
}
use paymentsv1::payments_server::{Payments, PaymentsServer};
use paymentsv1::{LogAndSettleRequest, LogAndSettleResponse};

mod idempo;
use idempo::IdempoCache;

// ========== Interceptor untuk ambil idempotency-key ==========
#[derive(Clone)]
struct IdempoInterceptor;
impl Interceptor for IdempoInterceptor {
    fn call(&mut self, mut req: Request<()>) -> Result<Request<()>, Status> {
        // hanya meneruskan; bila perlu validasi metadata di sini
        Ok(req)
    }
}

// helper: ambil idempotency-key dari metadata atau field req
fn get_idempo_from_meta(meta: &MetadataMap) -> Option<String> {
    if let Some(v) = meta.get("idempotency-key") {
        return Some(v.to_str().ok()?.to_string());
    }
    None
}

// ========== Payments service ==========
#[derive(Clone)]
struct PaymentsSvc {
    db_addr: String,
    cache: IdempoCache,
}

#[tonic::async_trait]
impl Payments for PaymentsSvc {
    async fn log_and_settle(
        &self,
        req: Request<LogAndSettleRequest>
    ) -> Result<Response<LogAndSettleResponse>, Status> {
        // 1) idempotency-key
        let meta_key = get_idempo_from_meta(req.metadata());
        let mut r = req.into_inner();
        let key = r.idempotency_key.clone().or(meta_key).unwrap_or_else(|| Uuid::new_v4().to_string());
        r.idempotency_key = Some(key.clone());

        // 2) fast-replay cache
        if let Some(hit) = self.cache.get(&key).await {
            // bisa decode json; di sini cukup flag replay:
            return Ok(Response::new(LogAndSettleResponse {
                status: "SUCCESS_REPLAY".into(),
                message: hit,
                reservation_id: "".into(),
            }));
        }

        // 3) call db-rs: Reserve
        let mut db = DbClient::connect(format!("http://{}", self.db_addr))
            .await.map_err(|e| Status::unavailable(e.to_string()))?;
        let reserve = db.reserve_funds(ReserveFundsRequest{
            idempotency_key: key.clone(),
            sender_id: r.sender_id.clone(),
            receiver_id: r.receiver_id.clone(),
            amount_idr: r.amount_idr,           // pastikan sudah IDR-converted di Wallet stage
            currency_input: r.currency.clone(), // simpan untuk log
            created_at_iso: r.tx_date.clone(),
        }).await.map_err(|e| Status::internal(e.to_string()))?.into_inner();

        match reserve.status {
            x if x == TxStatus::TX_STATUS_INSUFFICIENT as i32 => {
                let res = LogAndSettleResponse{ status:"FAILED".into(), message:"insufficient_funds".into(), reservation_id:"".into() };
                self.cache.put(key.clone(), res.message.clone()).await;
                return Ok(Response::new(res));
            }
            x if x == TxStatus::TX_STATUS_DUPLICATE as i32 => {
                // Duplicate reserve: treat as replay success path – try commit anyway or just return SUCCESS_REPLAY
                let res = LogAndSettleResponse{ status:"SUCCESS_REPLAY".into(), message:"duplicate_idempotency_key".into(), reservation_id:"".into() };
                self.cache.put(key.clone(), res.message.clone()).await;
                return Ok(Response::new(res));
            }
            x if x != TxStatus::TX_STATUS_OK as i32 => {
                return Err(Status::internal("reserve_error"));
            }
            _ => {}
        }

        // 4) Commit
        let commit = db.commit_reservation(CommitReservationRequest{
            reservation_id: reserve.reservation_id.clone(),
            idempotency_key: key.clone(),
        }).await.map_err(|e| Status::internal(e.to_string()))?.into_inner();

        if commit.status != TxStatus::TX_STATUS_OK as i32 {
            // gagal komit → rollback untuk safety
            let _ = db.rollback_reservation(RollbackReservationRequest{
                reservation_id: reserve.reservation_id.clone(),
                reason: "commit_failed".into(),
            }).await;
            return Err(Status::internal("commit_failed"));
        }

        let res = LogAndSettleResponse{
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

    let db_addr = std::env::var("DB_ADDR").unwrap_or_else(|_| "db-rs:9095".into());
    let addr = "0.0.0.0:9096".parse().unwrap();

    let svc = PaymentsSvc {
        db_addr,
        cache: IdempoCache::new(),
    };

    // /metrics sederhana
    tokio::spawn(async move {
        use prometheus::{Registry, TextEncoder, Encoder};
        use hyper::{Body, Request as HReq, Response as HRes, Server as HServer};
        let registry = Registry::new();
        let make = hyper::service::make_service_fn(move |_| {
            let registry = registry.clone();
            async move {
                Ok::<_, hyper::Error>(hyper::service::service_fn(move |_req: HReq<Body>| {
                    let registry = registry.clone();
                    async move {
                        let mut buf = Vec::new();
                        let enc = TextEncoder::new();
                        enc.encode(&registry.gather(), &mut buf).unwrap();
                        Ok::<_, hyper::Error>(HRes::new(Body::from(buf)))
                    }
                }))
            }
        });
        let _ = HServer::bind(&"0.0.0.0:9106".parse().unwrap()).serve(make).await;
    });

    info!("payments-rs gRPC on {}", addr);
    Server::builder()
        .layer(tonic::service::interceptor(IdempoInterceptor))
        .add_service(PaymentsServer::new(svc))
        .serve(addr)
        .await?;

    Ok(())
}
