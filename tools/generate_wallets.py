#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
Generate dummy wallets to PostgreSQL with requested balance distribution.

Usage:
  python3 generate_wallets.py --n 1000 \
    --dsn postgresql://postgres:secret@localhost:5432/poc \
    --start-id 1000000000 \
    --schema public \
    --table wallet_accounts

Defaults:
  --dsn        postgresql://postgres:secret@localhost:5432/poc
  --start-id   1000000000
  --schema     public
  --table      wallet_accounts
"""

import argparse
import math
import random
from datetime import datetime, timezone

import psycopg2
import psycopg2.extras

def parse_args():
    p = argparse.ArgumentParser()
    p.add_argument("--n", type=int, default=1000, help="Jumlah akun yang akan dibuat")
    p.add_argument("--dsn", type=str, default="postgresql://postgres:secret@localhost:5432/poc", help="PostgreSQL DSN")
    p.add_argument("--schema", type=str, default="public", help="Nama schema PostgreSQL")
    p.add_argument("--table", type=str, default="wallet_accounts", help="Nama tabel")
    p.add_argument("--start-id", type=int, default=1000000000, help="Account id awal (10 digit)")
    p.add_argument("--truncate", action="store_true", help="Kosongkan tabel sebelum insert")
    p.add_argument("--seed", type=int, default=42, help="Random seed")
    return p.parse_args()

def ensure_schema(conn, schema, table):
    with conn.cursor() as cur:
        # Optional: pg_trgm untuk index GIN (jika tak ada izin, boleh diabaikan)
        try:
            cur.execute("CREATE EXTENSION IF NOT EXISTS pg_trgm;")
        except Exception:
            conn.rollback()  # lanjut tanpa pg_trgm
        else:
            conn.commit()

        cur.execute(f"""
        CREATE SCHEMA IF NOT EXISTS {schema};
        """)
        cur.execute(f"""
        CREATE TABLE IF NOT EXISTS {schema}.{table} (
            account_id   BIGINT PRIMARY KEY,
            account_name TEXT NOT NULL,
            balance_idr  BIGINT NOT NULL CHECK (balance_idr >= 0),
            last_updated TIMESTAMPTZ NOT NULL DEFAULT NOW()
        );
        """)
        cur.execute(f"CREATE INDEX IF NOT EXISTS idx_{table}_balance ON {schema}.{table} (balance_idr);")
        # Buat GIN trgm kalau extension aktif
        try:
            cur.execute(f"CREATE INDEX IF NOT EXISTS idx_{table}_name_trgm ON {schema}.{table} USING GIN (account_name gin_trgm_ops);")
        except Exception:
            conn.rollback()  # skip jika tidak bisa
        conn.commit()

def truncate_table(conn, schema, table):
    with conn.cursor() as cur:
        cur.execute(f"TRUNCATE TABLE {schema}.{table};")
    conn.commit()

def rand_amount(lo, hi):
    # Inclusive range in IDR, integer rupiah
    return random.randint(lo, hi)

def build_balances(n):
    """
    Distribusi:
      - 70%:   15.000  s/d  9.999.999
      - 20%: 10.000.000 s/d 49.999.999
      - 10%: 50.000.000 s/d 150.000.000
    """
    n_70 = int(round(n * 0.70))
    n_20 = int(round(n * 0.20))
    # Sisa ke 10%
    n_10 = n - n_70 - n_20

    # Koreksi jika pembulatan membuat negatif/kelebihan
    if n_10 < 0:
        n_10 = 0
        n_20 = max(0, n - n_70)

    low = [rand_amount(15_000, 9_999_999) for _ in range(n_70)]
    mid = [rand_amount(10_000_000, 49_999_999) for _ in range(n_20)]
    high = [rand_amount(50_000_000, 150_000_000) for _ in range(n_10)]

    arr = low + mid + high
    random.shuffle(arr)
    return arr, (len(low), len(mid), len(high))

def upsert_wallets(conn, schema, table, rows):
    # rows: list of dict
    cols = ["account_id", "account_name", "balance_idr", "last_updated"]
    with conn.cursor() as cur:
        psycopg2.extras.execute_values(
            cur,
            f"""
            INSERT INTO {schema}.{table} ({", ".join(cols)})
            VALUES %s
            ON CONFLICT (account_id) DO UPDATE SET
              account_name = EXCLUDED.account_name,
              balance_idr  = EXCLUDED.balance_idr,
              last_updated = EXCLUDED.last_updated;
            """,
            [
                (r["account_id"], r["account_name"], r["balance_idr"], r["last_updated"])
                for r in rows
            ],
            page_size=500
        )
    conn.commit()

def main():
    args = parse_args()
    random.seed(args.seed)

    # Validasi 10 digit untuk start-id
    if len(str(args.start_id)) != 10:
        raise SystemExit("--start-id harus 10 digit, contoh: 1000000000")

    conn = psycopg2.connect(args.dsn)
    try:
        ensure_schema(conn, args.schema, args.table)
        if args.truncate:
            truncate_table(conn, args.schema, args.table)

        balances, parts = build_balances(args.n)
        base = args.start_id

        now = datetime.now(timezone.utc)
        rows = []
        for i in range(args.n):
            acc_id = base + i  # tetap 10 digit selama tidak melewati 9.999.999.999
            rows.append({
                "account_id": acc_id,
                "account_name": f"Customer {i+1}",
                "balance_idr": balances[i],
                "last_updated": now
            })

        upsert_wallets(conn, args.schema, args.table, rows)

        n70, n20, n10 = parts
        print(f"✅ Insert/Upsert selesai: {args.n} akun")
        print(f"   Distribusi: 70%(<10jt)={n70}, 20%(10–50jt)={n20}, 10%(50–150jt)={n10}")
        print(f"   Tabel: {args.schema}.{args.table} | DSN: {args.dsn}")
    finally:
        conn.close()

if __name__ == "__main__":
    main()
