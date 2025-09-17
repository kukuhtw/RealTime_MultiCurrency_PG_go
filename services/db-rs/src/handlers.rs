//services/db-rs/src/handlers.rs

use tonic::{Request, Response, Status};
use uuid::Uuid;

use crate::dbv1::db_server::Db;
use crate::dbv1::*;
use crate::store;

#[derive(Clone)]
pub struct DbService {
    pool: sqlx::PgPool,
}

impl DbService {
    pub fn new(pool: sqlx::PgPool) -> Self {
        Self { pool }
    }
}

#[tonic::async_trait]
impl Db for DbService {
    async fn reserve_funds(
        &self,
        req: Request<ReserveFundsRequest>,
    ) -> Result<Response<ReserveFundsResponse>, Status> {
        let r = req.into_inner();

        // basic validations
        if r.idempotency_key.is_empty()
            || r.sender_id.is_empty()
            || r.receiver_id.is_empty()
            || r.amount_idr <= 0
        {
            return Err(Status::invalid_argument("bad_input"));
        }

        // begin tx
        let mut tx = self.pool.begin().await.map_err(internal)?;

        match store::reserve_funds(
            &mut tx,
            &r.idempotency_key,
            &r.sender_id,
            &r.receiver_id,
            r.amount_idr,
            &r.currency_input,
        )
        .await
        {
            Ok(store::ReserveResult::Duplicate { reservation_id }) => {
                tx.commit().await.map_err(internal)?;
                Ok(Response::new(ReserveFundsResponse {
                    status: TxStatus::TxStatusDuplicate as i32,
                    reservation_id: reservation_id.to_string(),
                }))
            }
            Ok(store::ReserveResult::Insufficient) => {
                tx.rollback().await.map_err(internal)?;
                Ok(Response::new(ReserveFundsResponse {
                    status: TxStatus::TxStatusInsufficient as i32,
                    reservation_id: "".into(),
                }))
            }
            Ok(store::ReserveResult::Ok { reservation_id }) => {
                tx.commit().await.map_err(internal)?;
                Ok(Response::new(ReserveFundsResponse {
                    status: TxStatus::TxStatusOk as i32,
                    reservation_id: reservation_id.to_string(),
                }))
            }
            Err(e) => {
                let _ = tx.rollback().await;
                Err(internal(e))
            }
        }
    }

    async fn commit_reservation(
        &self,
        req: Request<CommitReservationRequest>,
    ) -> Result<Response<CommitReservationResponse>, Status> {
        let r = req.into_inner();

        let rid = Uuid::parse_str(&r.reservation_id)
            .map_err(|_| Status::invalid_argument("bad_reservation_id"))?;

        let mut tx = self.pool.begin().await.map_err(internal)?;

        match store::commit_reservation(&mut tx, rid, &r.idempotency_key).await {
            Ok(store::CommitResult::Ok) => {
                tx.commit().await.map_err(internal)?;
                Ok(Response::new(CommitReservationResponse {
                    status: TxStatus::TxStatusOk as i32,
                }))
            }
            Ok(store::CommitResult::ReplayOk) => {
                tx.commit().await.map_err(internal)?;
                Ok(Response::new(CommitReservationResponse {
                    status: TxStatus::TxStatusOk as i32,
                }))
            }
            Ok(store::CommitResult::NotFound) => {
                tx.rollback().await.map_err(internal)?;
                Ok(Response::new(CommitReservationResponse {
                    status: TxStatus::TxStatusNotFound as i32,
                }))
            }
            Ok(store::CommitResult::BadStatus) => {
                tx.rollback().await.map_err(internal)?;
                Ok(Response::new(CommitReservationResponse {
                    status: TxStatus::TxStatusBadStatus as i32,
                }))
            }
            Err(e) => {
                let _ = tx.rollback().await;
                Err(internal(e))
            }
        }
    }

    async fn rollback_reservation(
        &self,
        req: Request<RollbackReservationRequest>,
    ) -> Result<Response<RollbackReservationResponse>, Status> {
        let r = req.into_inner();
        let rid = Uuid::parse_str(&r.reservation_id)
            .map_err(|_| Status::invalid_argument("bad_reservation_id"))?;

        let mut tx = self.pool.begin().await.map_err(internal)?;

        match store::rollback_reservation(&mut tx, rid, &r.reason).await {
            Ok(store::RollbackResult::Ok) => {
                tx.commit().await.map_err(internal)?;
                Ok(Response::new(RollbackReservationResponse {
                    status: TxStatus::TxStatusOk as i32,
                }))
            }
            Ok(store::RollbackResult::NotFound) => {
                tx.rollback().await.map_err(internal)?;
                Ok(Response::new(RollbackReservationResponse {
                    status: TxStatus::TxStatusNotFound as i32,
                }))
            }
            Ok(store::RollbackResult::BadStatus) => {
                tx.rollback().await.map_err(internal)?;
                Ok(Response::new(RollbackReservationResponse {
                    status: TxStatus::TxStatusBadStatus as i32,
                }))
            }
            Err(e) => {
                let _ = tx.rollback().await;
                Err(internal(e))
            }
        }
    }
}

fn internal<E: std::fmt::Display>(e: E) -> Status {
    Status::internal(e.to_string())
}
