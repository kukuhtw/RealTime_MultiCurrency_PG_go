#!/usr/bin/env python3
import argparse, random, json, time
import psycopg2, requests

def get_accounts(conn):
    with conn.cursor() as c:
        c.execute("select account_id from wallet_accounts limit 1000")
        rows = c.fetchall()
    ids = [r[0] for r in rows]
    return ids

def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--n", type=int, default=100)
    ap.add_argument("--dsn", required=True, help="postgresql://postgres:secret@localhost:5432/poc")
    ap.add_argument("--api", default="http://localhost:18080/api/payments")
    args = ap.parse_args()

    conn = psycopg2.connect(args.dsn)
    ids = get_accounts(conn)
    cur_choices = ["IDR","USD","SGD"]

    for i in range(args.n):
        sender, receiver = random.sample(ids, 2)
        payload = {
            "sender_id": sender,
            "receiver_id": receiver,
            "currency": random.choice(cur_choices),
            "amount": round(random.uniform(1.0, 5_000_000.0), 2),
            "tx_date": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
            "idempotency_key": f"bulk-{int(time.time()*1000)}-{i}"
        }
        try:
            r = requests.post(args.api, json=payload, timeout=5)
            print(i, r.status_code, r.text[:200])
        except Exception as e:
            print("ERR", i, e)

if __name__ == "__main__":
    main()
