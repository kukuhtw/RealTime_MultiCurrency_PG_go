// tests/e2e/payment_grpc_test.js

import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';

const client = new grpc.Client();
client.load(['proto'], 'payments/v1/payments.proto');

export const options = {
  vus: 5,          // jumlah virtual users
  duration: '30s', // lama load test
};

export default function () {
  // koneksi ke service payments-grpc (alamat harus sesuai docker-compose)
  client.connect('payments-grpc:9091', {
    plaintext: true,
  });

  const data = {
    tx: { tx_id: `TX-${__VU}-${Date.now()}` },
    customer: { customer_id: `CUST-${__VU}` },
    account: { account_id: 'ACC-1' },
    amount: { currency: 1, amount: 100 }, // USD
    settlement_currency: 2, // IDR
    merchant_id: 'MRC-123',
    ip: '127.0.0.1',
    device_id: 'DEV-abc',
    billing_country: 'ID',
    mcc: '5999',
  };

  const response = client.invoke(
    'payments.v1.PaymentsService/CreatePayment',
    data
  );

  check(response, {
    'status is OK': (r) => r && r.status === grpc.StatusOK,
    'tx_id returned': (r) => r && r.message && r.message.tx_id !== '',
    'payment not failed': (r) =>
      r && r.message && r.message.status !== 'PAYMENT_STATUS_FAILED',
  });

  client.close();

  sleep(1);
}
