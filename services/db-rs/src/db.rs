// db-rs/src/db.rs

use sqlx::{PgPool, Postgres, Transaction};
use uuid::Uuid;
use anyhow::Result;

pub async fn reserve_funds(
    tx: &mut Transaction<'_, Postgres>,
    idempotency_key: &str,
    sender_id: &str,
    receiver_id: &str,
    amount_idr: i64,
    currency_input: &str,
) -> Result<(Uuid, String)> {
    // 1. cek duplicate
    if let Some(row) = sqlx::query!(
        "SELECT reservation_id, status FROM reservations WHERE idempotency_key=$1",
        idempotency_key
    )
    .fetch_optional(&mut *tx)
    .await?
    {
        return Ok((row.reservation_id, "DUPLICATE".to_string()));
    }

    // 2. lock saldo sender
    let rec = sqlx::query!(
        "SELECT balance_idr FROM wallet_accounts WHERE account_id=$1 FOR UPDATE",
        sender_id
    )
    .fetch_one(&mut *tx)
    .await?;

    if rec.balance_idr < amount_idr {
        return Ok((Uuid::nil(), "INSUFFICIENT".to_string()));
    }

    // 3. kurangi saldo
    sqlx::query!(
        "UPDATE wallet_accounts SET balance_idr=balance_idr-$1, updated_at=now() WHERE account_id=$2",
        amount_idr,
        sender_id
    )
    .execute(&mut *tx)
    .await?;

    // 4. insert reservation
    let rid = Uuid::new_v4();
    sqlx::query!(
        "INSERT INTO reservations (reservation_id,idempotency_key,sender_id,receiver_id,amount_idr,currency_input,status)
         VALUES ($1,$2,$3,$4,$5,$6,'PENDING')",
        rid, idempotency_key, sender_id, receiver_id, amount_idr, currency_input
    )
    .execute(&mut *tx)
    .await?;

    Ok((rid, "OK".to_string()))
}
