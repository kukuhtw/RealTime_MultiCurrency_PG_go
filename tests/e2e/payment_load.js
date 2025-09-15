// tests/e2e/payment_load.js
import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

// Base URL dibaca dari ENV (aman untuk docker)
const API_GATEWAY = __ENV.API_GATEWAY || 'http://localhost:8080';
const PAYMENTS    = __ENV.PAYMENTS    || 'http://localhost:8081';
const FX          = __ENV.FX          || 'http://localhost:8082';
const WALLET      = __ENV.WALLET      || 'http://localhost:8083';
const RISK        = __ENV.RISK        || 'http://localhost:8084';

export const options = {
  stages: [
    { duration: '20s', target: 5 },
    { duration: '40s', target: 10 },
    { duration: '20s', target: 0 },
  ],
  thresholds: {
    http_req_failed:   ['rate<0.05'],   // < 5% gagal di sisi k6
    http_req_duration: ['p(95)<500'],   // p95 < 500ms (sesuaikan)
  },
  tags: { test: 'payment-load' },
};

function acc() { return `ACC${randomIntBetween(10000, 99999)}`; }
function pick(a) { return a[randomIntBetween(0, a.length - 1)]; }

export default function () {
  // FX: rate
  const pairs = [['USD','IDR'], ['EUR','USD'], ['IDR','USD']];
  let [base, quote] = pick(pairs);
  let res = http.get(`${FX}/rate?base=${base}&quote=${quote}`);
  check(res, { 'fx 200': r => r.status === 200 });

  // Wallet: balance
  res = http.get(`${WALLET}/balance/${acc()}`);
  check(res, { 'wallet 200': r => r.status === 200 });

  // Risk: score
  const riskBody = JSON.stringify({ account: acc(), amount: randomIntBetween(1, 1000), currency: 'IDR' });
  res = http.post(`${RISK}/score`, riskBody, { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'risk 200': r => r.status === 200 });

  // Payments: allow 10% 500 (sesuai service kamu)
  const payBody = JSON.stringify({
    id: `PAY-${Date.now()}-${__VU}-${__ITER}`,
    currency: 'IDR',
    amount: randomIntBetween(1, 100),
    source_account: acc(),
    destination_account: acc(),
  });
  res = http.post(`${PAYMENTS}/payments`, payBody, { headers: { 'Content-Type': 'application/json' } });
  check(res, { 'payments 2xx/5xx': r => r.status === 200 || r.status === 500 });

  sleep(1);
}
