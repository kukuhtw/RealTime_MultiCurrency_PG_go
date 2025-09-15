// tests/e2e/payment_load_from_csv.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { SharedArray } from 'k6/data';
import papaparse from 'https://jslib.k6.io/papaparse/5.1.1/index.js';

// ====== sumber data (parametrik) ======
const PAYMENTS = __ENV.PAYMENTS || 'http://localhost:8081';
//const CSV_PATH = __ENV.CSV_PATH || './tests/data/dummy_transactions.csv';
const CSV_PATH = __ENV.CSV_PATH || '../data/dummy_transactions.csv';

// Load & filter CSV sekali untuk semua VU
const csvData = new SharedArray('payment data', () => {
  const parsed = papaparse.parse(open(CSV_PATH), {
    header: true,
    skipEmptyLines: true,
  }).data;

  // hanya baris yang lengkap
  return parsed.filter(r =>
    r.id &&
    r.currency &&
    r.amount &&
    r.source_account &&
    r.destination_account
  );
});

export const options = {
  stages: [
    { duration: '30s', target: 10 },
    { duration: '1m',  target: 10 },
    { duration: '30s', target: 0  },
  ],
};

export default function () {
  const row = csvData[Math.floor(Math.random() * csvData.length)];

  const url = `${PAYMENTS}/payments`;
  const payload = JSON.stringify({
    id: row.id,
    currency: row.currency,
    amount: parseFloat(row.amount), // pastikan numeric
    source_account: row.source_account,
    destination_account: row.destination_account,
  });

  const res = http.post(url, payload, { headers: { 'Content-Type': 'application/json' } });

  // NOTE:
  // service payments kamu sekarang mengembalikan {"status":"ok"}.
  // Kalau ingin cek 'id', ubah handler agar me-reply {"status":"ok","id":...}
  check(res, {
    'status is 200': (r) => r.status === 200,
    // 'response has id': (r) => r.json('id') !== undefined, // aktifkan jika server echo id
  });

  sleep(1);
}
