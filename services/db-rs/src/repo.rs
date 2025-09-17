// services/db-rs/src/repo.rs

use sqlx::{PgPool, Postgres, Transaction};
use uuid::Uuid;

pub struct Repo {
    pub pool: PgPool,
}

impl Repo {
    pub async fn check_balance(&self, account_id: &str, amount: i64) -> sqlx::Result<(bool,i64)> {
        let row = sqlx::query!("SELECT balance_idr FROM wallet_accounts WHERE account_id = $1", account_id)
            .fetch_one(&self.pool)
            .await?;
        let sufficient = row.balance_idr >= amount;
        Ok((sufficient, row.balance_idr))
    }

    pub async fn reserve_funds(
        &self,
        idempotency_key: &str,
        sender_id: &str,
        receiver_id: &str,
        amount_idr: i64
    ) -> sqlx::Result<(Uuid, i64)> {
        let mut tx: Transaction<'_, Postgres> = self.pool.begin().await?;

        // Idempotency check – if key exists, treat as duplicate reserve
        if let Some(existing) = sqlx::query!("SELECT reservation_id FROM reservations WHERE idempotency_key = $1", idempotency_key)
            .fetch_optional(&mut *tx).await? {
            let bal = sqlx::query!("SELECT balance_idr FROM wallet_accounts WHERE account_id=$1", sender_id)
                .fetch_one(&mut *tx).await?.balance_idr;
            tx.commit().await?;
            return Ok((existing.reservation_id, bal));
        }

        // lock sender
        let sender = sqlx::query!("SELECT balance_idr FROM wallet_accounts WHERE account_id = $1 FOR UPDATE", sender_id)
            .fetch_one(&mut *tx).await?;
        if sender.balance_idr < amount_idr {
            // insert reservation row but we could also skip; here we skip inserting for insufficient
            tx.rollback().await?;
            anyhow::bail!("INSUFFICIENT");
        }

        // debit sementara
        let new_bal = sender.balance_idr - amount_idr;
        sqlx::query!("UPDATE wallet_accounts SET balance_idr=$1, updated_at=now() WHERE account_id=$2",
            new_bal, sender_id).execute(&mut *tx).await?;

        // simpan reservation PENDING
        let rid = Uuid::new_v4();
        sqlx::query!(
            "INSERT INTO reservations (reservation_id, idempotency_key, sender_id, receiver_id, amount_idr, status)
             VALUES ($1,$2,$3,$4,$5,'PENDING')",
            rid, idempotency_key, sender_id, receiver_id, amount_idr
        ).execute(&mut *tx).await?;

        tx.commit().await?;
        Ok((rid, new_bal))
    }

    pub async fn commit_reservation(&self, reservation_id: Uuid, idempotency_key: &str) -> sqlx::Result<()> {
        let mut tx = self.pool.begin().await?;

        // get reservation
        let res = sqlx::query!("SELECT sender_id, receiver_id, amount_idr, status FROM reservations WHERE reservation_id = $1 FOR UPDATE", reservation_id)
            .fetch_one(&mut *tx).await?;
        if res.status == "COMMITTED" {
            tx.commit().await?;
            return Ok(());
        }
        if res.status != "PENDING" {
            tx.rollback().await?;
            anyhow::bail!("INVALID_STATE");
        }

        // credit receiver
        let recv = sqlx::query!("SELECT balance_idr FROM wallet_accounts WHERE account_id = $1 FOR UPDATE", res.receiver_id)
            .fetch_one(&mut *tx).await?;
        let new_recv_bal = recv.balance_idr + res.amount_idr;
        sqlx::query!("UPDATE wallet_accounts SET balance_idr=$1, updated_at=now() WHERE account_id=$2",
            new_recv_bal, res.receiver_id).execute(&mut *tx).await?;

        // mark reservation committed
        sqlx::query!("UPDATE reservations SET status='COMMITTED' WHERE reservation_id=$1", reservation_id)
            .execute(&mut *tx).await?;

        // write payments (success) – unique on idempotency_key
        sqlx::query!(
            "INSERT INTO payments (payment_id,idempotency_key,sender_id,receiver_id,currency_input,amount_idr,status)
             VALUES ($1,$2,$3,$4,$5,$6,'SUCCESS')
             ON CONFLICT (idempotency_key) DO NOTHING",
            Uuid::new_v4(), idempotency_key, res.sender_id, res.receiver_id, "IDR", res.amount_idr
        ).execute(&mut *tx).await?;

        tx.commit().await?;
        Ok(())
    }

    pub async fn rollback_reservation(&self, reservation_id: Uuid, reason: &str) -> sqlx::Result<()> {
        let mut tx = self.pool.begin().await?;
        let res = sqlx::query!("SELECT sender_id, amount_idr, status FROM reservations WHERE reservation_id=$1 FOR UPDATE", reservation_id)
            .fetch_one(&mut *tx).await?;

        if res.status != "PENDING" {
            tx.rollback().await?;
            anyhow::bail!("INVALID_STATE");
        }

        // return debit sementara
        let sender = sqlx::query!("SELECT balance_idr FROM wallet_accounts WHERE account_id=$1 FOR UPDATE", res.sender_id)
            .fetch_one(&mut *tx).await?;
        let restored = sender.balance_idr + res.amount_idr;
        sqlx::query!("UPDATE wallet_accounts SET balance_idr=$1, updated_at=now() WHERE account_id=$2",
            restored, res.sender_id).execute(&mut *tx).await?;

        sqlx::query!("UPDATE reservations SET status='ROLLEDBACK' WHERE reservation_id=$1", reservation_id)
            .execute(&mut *tx).await?;

        // write payments (failed)
        sqlx::query!(
            "INSERT INTO payments (payment_id,idempotency_key,sender_id,receiver_id,currency_input,amount_idr,status)
             VALUES ($1, concat('rb-', $2), $3, '', 'IDR', 0, 'FAILED')",
            uuid::Uuid::new_v4(), reason, res.sender_id
        ).execute(&mut *tx).await?;

        tx.commit().await?;
        Ok(())
    }

    pub async fn get_random_accounts(&self) -> sqlx::Result<(String,String)> {
        let row = sqlx::query!("SELECT account_id FROM wallet_accounts ORDER BY random() LIMIT 2")
            .fetch_all(&self.pool).await?;
        let a = row.get(0).unwrap().account_id.clone();
        let b = row.get(1).unwrap().account_id.clone();
        Ok((a,b))
    }
}
