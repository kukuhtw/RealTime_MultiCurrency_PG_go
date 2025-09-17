// services/db-rs/src/store.rs

use anyhow::{anyhow, Result};
use sqlx::{Postgres, Transaction};
use uuid::Uuid;

#[derive(Debug)]
pub enum ReserveResult {
    Ok { reservation_id: Uuid, current_balance: i64 },
    Insufficient { current_balance: i64 },
    Duplicate { reservation_id: Uuid, current_balance: i64 },
}


#[derive(Debug)]
pub enum CommitResult {
    Ok,
    ReplayOk,
    NotFound,
    BadStatus,
}

#[derive(Debug)]
pub enum RollbackResult {
    Ok,
    NotFound,
    BadStatus,
}

/// ReserveFunds:
/// - Idempotent dengan idempotency_key (cek di reservations)
/// - Lock saldo sender (FOR UPDATE), cek cukup, kurangi, insert reservation PENDING
// services/db-rs/src/store.rs

pub async fn reserve_funds(
    tx: &mut Transaction<'_, Postgres>,
    idempotency_key: &str,
    sender_id: &str,
    receiver_id: &str,
    amount_idr: i64,
    currency_input: &str,
) -> Result<ReserveResult> {
    // 0) Duplicate?
    if let Some(row) = sqlx::query!(
        r#"SELECT reservation_id, status
           FROM reservations
           WHERE idempotency_key = $1
           FOR UPDATE"#,
        idempotency_key
    )
    .fetch_optional(&mut **tx)
    .await?
    {
        // ambil saldo terkini sender untuk response
        let s = sqlx::query!(
            r#"SELECT balance_idr FROM wallet_accounts WHERE account_id = $1 FOR UPDATE"#,
            sender_id
        )
        .fetch_one(&mut **tx)
        .await?;
        return Ok(ReserveResult::Duplicate {
            reservation_id: row.reservation_id,
            current_balance: s.balance_idr,
        });
    }

    // 1) Lock sender balance
    let sender = sqlx::query!(
        r#"SELECT balance_idr
           FROM wallet_accounts
           WHERE account_id = $1
           FOR UPDATE"#,
        sender_id
    )
    .fetch_optional(&mut **tx)
    .await?
    .ok_or_else(|| anyhow!("sender_not_found"))?;

    if sender.balance_idr < amount_idr {
        return Ok(ReserveResult::Insufficient {
            current_balance: sender.balance_idr,
        });
    }

    // hitung saldo baru lebih dulu agar bisa direturn
    let new_balance = sender.balance_idr - amount_idr;

    // 2) Deduct (hold)
    sqlx::query!(
        r#"UPDATE wallet_accounts
           SET balance_idr = $1, updated_at = now()
           WHERE account_id = $2"#,
        new_balance,
        sender_id
    )
    .execute(&mut **tx)
    .await?;

    // 3) Insert reservation
    let rid = Uuid::new_v4();
    sqlx::query!(
        r#"INSERT INTO reservations(
                reservation_id, idempotency_key,
                sender_id, receiver_id, amount_idr,
                currency_input, status
           ) VALUES ($1,$2,$3,$4,$5,$6,'PENDING')"#,
        rid,
        idempotency_key,
        sender_id,
        receiver_id,
        amount_idr,
        currency_input
    )
    .execute(&mut **tx)
    .await?;

    Ok(ReserveResult::Ok {
        reservation_id: rid,
        current_balance: new_balance,
    })
}

/// CommitReservation:
/// - Idempotent via payments.idempotency_key (jika sudah ada, anggap OK / replay)
/// - Kunci reservation (FOR UPDATE), harus PENDING
/// - Kredit receiver, set reservation -> COMMITTED, insert payments SUCCESS
pub async fn commit_reservation(
    tx: &mut Transaction<'_, Postgres>,
    reservation_id: Uuid,
    idempotency_key: &str,
) -> Result<CommitResult> {
    // replay?
    if let Some(_p) = sqlx::query!(
        r#"SELECT payment_id FROM payments
           WHERE idempotency_key = $1
           FOR UPDATE"#,
        idempotency_key
    )
    .fetch_optional(&mut **tx)
    .await?
    {
        return Ok(CommitResult::ReplayOk);
    }

    // load reservation
    let res = sqlx::query!(
        r#"SELECT sender_id, receiver_id, amount_idr, currency_input, status
           FROM reservations
           WHERE reservation_id = $1
           FOR UPDATE"#,
        reservation_id
    )
    .fetch_optional(&mut **tx)
    .await?;

    let res = match res {
        Some(r) => r,
        None => return Ok(CommitResult::NotFound),
    };

    if res.status != "PENDING" {
        return Ok(CommitResult::BadStatus);
    }

    // kredit receiver
    sqlx::query!(
        r#"UPDATE wallet_accounts
           SET balance_idr = balance_idr + $1, updated_at = now()
           WHERE account_id = $2"#,
        res.amount_idr,
        res.receiver_id
    )
    .execute(&mut **tx)
    .await?;

    // update reservation -> COMMITTED
    sqlx::query!(
        r#"UPDATE reservations
           SET status = 'COMMITTED'
           WHERE reservation_id = $1"#,
        reservation_id
    )
    .execute(&mut **tx)
    .await?;

    // insert payments SUCCESS
    let pid = Uuid::new_v4();
    sqlx::query!(
        r#"INSERT INTO payments(
                payment_id, idempotency_key,
                sender_id, receiver_id, currency_input,
                amount_idr, status
           ) VALUES ($1,$2,$3,$4,$5,$6,'SUCCESS')"#,
        pid,
        idempotency_key,
        res.sender_id,
        res.receiver_id,
        res.currency_input,
        res.amount_idr
    )
    .execute(&mut **tx)
    .await?;

    Ok(CommitResult::Ok)
}

/// RollbackReservation:
/// - Kunci reservation PENDING, kredit balik sender,
/// - set reservation -> ROLLEDBACK, insert payments FAILED
pub async fn rollback_reservation(
    tx: &mut Transaction<'_, Postgres>,
    reservation_id: Uuid,
    reason: &str,
) -> Result<RollbackResult> {
    let res = sqlx::query!(
        r#"SELECT idempotency_key, sender_id, receiver_id, amount_idr, currency_input, status
           FROM reservations
           WHERE reservation_id = $1
           FOR UPDATE"#,
        reservation_id
    )
    .fetch_optional(&mut **tx)
    .await?;

    let res = match res {
        Some(r) => r,
        None => return Ok(RollbackResult::NotFound),
    };

    if res.status != "PENDING" {
        return Ok(RollbackResult::BadStatus);
    }

    // kredit balik sender
    sqlx::query!(
        r#"UPDATE wallet_accounts
           SET balance_idr = balance_idr + $1, updated_at = now()
           WHERE account_id = $2"#,
        res.amount_idr,
        res.sender_id
    )
    .execute(&mut **tx)
    .await?;

    // update reservation -> ROLLEDBACK
    sqlx::query!(
        r#"UPDATE reservations
           SET status = 'ROLLEDBACK'
           WHERE reservation_id = $1"#,
        reservation_id
    )
    .execute(&mut **tx)
    .await?;

    // insert payments FAILED
    let pid = Uuid::new_v4();
    let fail_reason = if reason.is_empty() { "failed" } else { reason };
    sqlx::query!(
        r#"INSERT INTO payments(
                payment_id, idempotency_key,
                sender_id, receiver_id, currency_input,
                amount_idr, status
           ) VALUES ($1,$2,$3,$4,$5,$6,'FAILED')"#,
        pid,
        res.idempotency_key,
        res.sender_id,
        res.receiver_id,
        res.currency_input,
        res.amount_idr
    )
    .execute(&mut **tx)
    .await?;

    // (opsional) kamu bisa tambah tabel payment_failures untuk simpan "reason"

    Ok(RollbackResult::Ok)
}
