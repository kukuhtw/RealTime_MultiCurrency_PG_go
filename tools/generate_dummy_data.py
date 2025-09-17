#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
Generate + Load dummy data ke Postgres (1 DB fisik: service `postgres`)
 - seeds/customers.json, seeds/wallet_accounts.json, seeds/fx_rates.json, seeds/risk_rules.json
 - seeds/transactions.csv
 - Insert ke tabel: customers, wallets, fx_rates, app_config(key='risk_rules'), transactions

Jalankan dari HOST/WSL:
  export DATABASE_URL='postgresql://postgres:secret@localhost:15432/poc'
  python3 tools/generate_dummy_data.py

Atau dari Compose service `data-loader` (yang kita tulis sebelumnya).
"""

import os, json, random, csv, datetime
from typing import List, Tuple

# ===== Kuantitas utama =====
NUM_CUSTOMERS = 1000
NUM_WALLETS   = 1000
NUM_TX        = 1000

# ===== Parameter distribusi =====
CURRENCIES    = ["USD", "IDR", "SGD"]
SAME_CUR_PCT  = 0.7   # 70% same-currency, 30% cross-currency
RANDOM_SEED   = 42
TX_WINDOW_SEC = 7 * 24 * 3600  # 7 hari terakhir

# ===== Output dirs =====
OUT_DIR_SEEDS = "seeds"
OUT_DIR_DATA  = "seeds"
os.makedirs(OUT_DIR_SEEDS, exist_ok=True)
os.makedirs(OUT_DIR_DATA,  exist_ok=True)

random.seed(RANDOM_SEED)

# ---------- 0) DB helpers ----------
DATABASE_URL = os.getenv("DATABASE_URL", "postgresql://postgres:secret@localhost:15432/poc")
USE_DB = True

try:
    import psycopg
    from psycopg.rows import tuple_row
except ImportError:
    print("psycopg belum terpasang. Menulis file seeds saja (tanpa insert DB).")
    USE_DB = False

def db_exec(schema_sql: str, upserts: dict, transactions_rows: List[Tuple]):
    if not USE_DB:
        print("Lewati DB insert (psycopg tidak tersedia).")
        return
    print(f"Connect DB: {DATABASE_URL}")
    with psycopg.connect(DATABASE_URL, autocommit=False) as conn:
        with conn.cursor(row_factory=tuple_row) as cur:
            # Buat schema (idempotent)
            cur.execute(schema_sql)

            # Upsert customers
            if upserts.get("customers"):
                cur.executemany("""
                    INSERT INTO customers (customer_id, name, email, phone)
                    VALUES (%s, %s, %s, %s)
                    ON CONFLICT (customer_id) DO UPDATE
                      SET name=EXCLUDED.name, email=EXCLUDED.email, phone=EXCLUDED.phone
                """, upserts["customers"])

            # Upsert wallets
            if upserts.get("wallets"):
                cur.executemany("""
                    INSERT INTO wallets (account_id, owner, currency, balance_minor, daily_limit_minor, status)
                    VALUES (%s, %s, %s, %s, %s, %s)
                    ON CONFLICT (account_id) DO UPDATE
                      SET owner=EXCLUDED.owner,
                          currency=EXCLUDED.currency,
                          balance_minor=EXCLUDED.balance_minor,
                          daily_limit_minor=EXCLUDED.daily_limit_minor,
                          status=EXCLUDED.status
                """, upserts["wallets"])

            # Upsert fx_rates
            if upserts.get("fx_rates"):
                cur.executemany("""
                    INSERT INTO fx_rates (base_currency, quote_currency, rate)
                    VALUES (%s, %s, %s)
                    ON CONFLICT (base_currency, quote_currency) DO UPDATE
                      SET rate=EXCLUDED.rate
                """, upserts["fx_rates"])

            # Upsert risk_rules ke app_config (JSON)
            if upserts.get("risk_rules"):
                cur.execute("""
                    INSERT INTO app_config (cfg_key, payload)
                    VALUES ('risk_rules', %s::jsonb)
                    ON CONFLICT (cfg_key) DO UPDATE
                      SET payload=EXCLUDED.payload
                """, (json.dumps(upserts["risk_rules"]),))

            # Insert transactions (idempotent dengan ON CONFLICT DO NOTHING)
            if transactions_rows:
                cur.executemany("""
                    INSERT INTO transactions
                      (id, source_account, destination_account, amount_minor,
                       currency_src, currency_dst, ts, description)
                    VALUES (%s,%s,%s,%s,%s,%s,%s,%s)
                    ON CONFLICT (id) DO NOTHING
                """, transactions_rows)

        conn.commit()
    print("DB insert: selesai.")

SCHEMA_SQL = """
CREATE TABLE IF NOT EXISTS customers (
  customer_id TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  email       TEXT NOT NULL,
  phone       TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS wallets (
  account_id        TEXT PRIMARY KEY,
  owner             TEXT NOT NULL REFERENCES customers(customer_id) ON DELETE RESTRICT,
  currency          TEXT NOT NULL,
  balance_minor     BIGINT NOT NULL,
  daily_limit_minor BIGINT NOT NULL,
  status            TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS fx_rates (
  base_currency  TEXT NOT NULL,
  quote_currency TEXT NOT NULL,
  rate           NUMERIC(18,6) NOT NULL,
  PRIMARY KEY (base_currency, quote_currency)
);

CREATE TABLE IF NOT EXISTS app_config (
  cfg_key TEXT PRIMARY KEY,
  payload JSONB NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
  id               TEXT PRIMARY KEY,
  source_account   TEXT NOT NULL REFERENCES wallets(account_id) ON DELETE RESTRICT,
  destination_account TEXT NOT NULL REFERENCES wallets(account_id) ON DELETE RESTRICT,
  amount_minor     BIGINT NOT NULL,
  currency_src     TEXT NOT NULL,
  currency_dst     TEXT NOT NULL,
  ts               TIMESTAMPTZ NOT NULL,
  description      TEXT
);
"""

# ---------- 1) Customers ----------
customers = []
for i in range(NUM_CUSTOMERS):
    cid = f"CUST-{i+1:04d}"
    customers.append({
        "customer_id": cid,
        "name": f"Customer {i+1:04d}",
        "email": f"cust{i+1:04d}@example.com",
        "phone": f"+628{random.randint(10000000, 99999999)}",
    })
with open(f"{OUT_DIR_SEEDS}/customers.json", "w", encoding="utf-8") as f:
    json.dump({"customers": customers}, f, indent=2, ensure_ascii=False)

# ---------- 2) Wallets ----------
wallets = []
for i in range(NUM_WALLETS):
    acc_id = f"ACC_{i+1:06d}"
    cur = random.choice(CURRENCIES)
    if cur == "IDR":
        balance_minor = int(random.uniform(5_000_00, 5_000_000_00))  # Rp 5jt – 500jt
    elif cur == "USD":
        balance_minor = int(random.uniform(50_00, 50_000_00))        # $50 – $50,000
    else:  # SGD
        balance_minor = int(random.uniform(50_00, 30_000_00))        # S$50 – S$30,000

    wallets.append({
        "account_id": acc_id,
        "owner": customers[i % NUM_CUSTOMERS]["customer_id"],
        "currency": cur,
        "balance_minor": balance_minor,
        "daily_limit_minor": balance_minor // 2,
        "status": "ACTIVE",
    })
with open(f"{OUT_DIR_SEEDS}/wallet_accounts.json", "w", encoding="utf-8") as f:
    json.dump({"accounts": wallets}, f, indent=2, ensure_ascii=False)

wallet_map = {w["account_id"]: w for w in wallets}

# ---------- 3) FX rates ----------
rates = [
    {"base_currency": "USD", "quote_currency": "IDR", "rate": 15500.00},
    {"base_currency": "SGD", "quote_currency": "IDR", "rate": 11500.00},
    {"base_currency": "IDR", "quote_currency": "USD", "rate": 0.000064},
    {"base_currency": "IDR", "quote_currency": "SGD", "rate": 0.000087},
    {"base_currency": "USD", "quote_currency": "SGD", "rate": 1.35},
    {"base_currency": "SGD", "quote_currency": "USD", "rate": 0.74},
]
with open(f"{OUT_DIR_SEEDS}/fx_rates.json", "w", encoding="utf-8") as f:
    json.dump({"rates": rates}, f, indent=2, ensure_ascii=False)

# ---------- 4) Risk rules ----------
risk_rules = {
    "max_amount": 100_000,             # major units (interpretasi server)
    "velocity_per_min": 200,
    "blocked_accounts": ["ACC_000999"],
    "blocked_countries": []
}
with open(f"{OUT_DIR_SEEDS}/risk_rules.json", "w", encoding="utf-8") as f:
    json.dump(risk_rules, f, indent=2, ensure_ascii=False)

# ---------- 5) Transactions CSV ----------
csv_path = f"{OUT_DIR_DATA}/transactions.csv"
fieldnames = [
    "id","source_account","destination_account",
    "amount_minor","currency_src","currency_dst",
    "timestamp","description",
]

def pick_pair_same_or_cross(src_acc_id: str):
    src_cur = wallet_map[src_acc_id]["currency"]
    dst_cur = src_cur if random.random() < SAME_CUR_PCT else random.choice([c for c in CURRENCIES if c != src_cur])
    return src_cur, dst_cur

def random_amount_minor(currency: str) -> int:
    if currency == "IDR":
        return int(random.uniform(50_000, 5_000_000))   # Rp 50rb – 5jt
    if currency == "USD":
        return int(random.uniform(100, 200_000))        # $1.00 – $2,000.00
    return int(random.uniform(100, 150_000))            # S$1.00 – S$1,500.00

def now_iso_utc_minus_rand(seconds_window: int) -> str:
    dt = datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(
        seconds=random.randint(0, seconds_window)
    )
    return dt.isoformat().replace("+00:00", "Z")

with open(csv_path, "w", newline="", encoding="utf-8") as csvf:
    writer = csv.DictWriter(csvf, fieldnames=fieldnames)
    writer.writeheader()
    for i in range(NUM_TX):
        txid = f"TX-{i+1:06d}"
        src = random.choice(wallets)["account_id"]
        dst = random.choice(wallets)["account_id"]
        while dst == src:
            dst = random.choice(wallets)["account_id"]
        cur_src, cur_dst = pick_pair_same_or_cross(src)
        amt_minor = random_amount_minor(cur_src)
        ts = now_iso_utc_minus_rand(TX_WINDOW_SEC)
        writer.writerow({
            "id": txid,
            "source_account": src,
            "destination_account": dst,
            "amount_minor": amt_minor,
            "currency_src": cur_src,
            "currency_dst": cur_dst,
            "timestamp": ts,
            "description": "poc-tx",
        })

# ---------- 6) Kumpulkan data untuk DB ----------
customers_rows = [(c["customer_id"], c["name"], c["email"], c["phone"]) for c in customers]
wallets_rows   = [(w["account_id"], w["owner"], w["currency"], w["balance_minor"], w["daily_limit_minor"], w["status"]) for w in wallets]
rates_rows     = [(r["base_currency"], r["quote_currency"], r["rate"]) for r in rates]

# Baca CSV kembali → rows untuk insert
tx_rows = []
with open(csv_path, newline="", encoding="utf-8") as f:
    r = csv.DictReader(f)
    for row in r:
        # parse ISO to Python datetime (psycopg akan kirim sebagai timestamptz)
        ts = datetime.datetime.fromisoformat(row["timestamp"].replace("Z","+00:00"))
        tx_rows.append((
            row["id"],
            row["source_account"],
            row["destination_account"],
            int(row["amount_minor"]),
            row["currency_src"],
            row["currency_dst"],
            ts,
            row["description"],
        ))

db_exec(
    SCHEMA_SQL,
    upserts={
        "customers": customers_rows,
        "wallets":   wallets_rows,
        "fx_rates":  rates_rows,
        "risk_rules": risk_rules,
    },
    transactions_rows=tx_rows
)

print("Generated:")
print(f" - seeds/customers.json         ({NUM_CUSTOMERS})")
print(f" - seeds/wallet_accounts.json   ({NUM_WALLETS})")
print(" - seeds/fx_rates.json")
print(" - seeds/risk_rules.json")
print(f" - seeds/transactions.csv       ({NUM_TX})")
print("Load ke Postgres: OK" if USE_DB else "Load ke Postgres: SKIPPED (psycopg not installed)")
