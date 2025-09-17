// tests/e2e/payment_grpc_test.js


import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';

const client = new grpc.Client();
// Perbaiki path proto - gunakan path yang benar relatif terhadap root project
client.load(['/work'], '/work/proto/gen/payments/v1/payments.proto');
export default function () {
  // Gunakan environment variable untuk target
  const target = __ENV.TARGET || 'payments-grpc:9091';
  client.connect(target, { plaintext: true });

  const req = {
    amount_minor: 1000,
    currency: 1, // USD enum value
    user_id: 'USER-123',
  };

  const res = client.invoke('payments.v1.PaymentsService/CreatePayment', req);
  check(res, { 
    'status is OK': (r) => r && r.status === grpc.StatusOK,
    'has payment_id': (r) => r && r.message && r.message.payment_id
  });

  client.close();
  sleep(1);
}