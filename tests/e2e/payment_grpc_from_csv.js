// tests/e2e/payment_grpc_from_csv.js
import grpc from 'k6/net/grpc';
import { check, sleep } from 'k6';
import papaparse from 'https://jslib.k6.io/papaparse/5.1.1/index.js';
import { SharedArray } from 'k6/data';

const client = new grpc.Client();

// Load proto files with correct paths
client.load(
  ['proto/gen'],  // Base directory for imports
  'common/v1/common.proto',
  'payments/v1/payments.proto'
);

const CSV_PATH = '../../work/data/dummy_transactions.csv';
const TARGET = 'payments-grpc:9091';

function toMinor(amountStr) {
  if (amountStr == null) return null;
  const cleaned = String(amountStr).trim().replace(/,/g, '').replace(/[^\d.\-]/g, '');
  if (cleaned === '' || cleaned === '.' || cleaned === '-') return null;
  const num = parseFloat(cleaned);
  if (!Number.isFinite(num)) return null;
  return Math.round(num * 100);
}

const rows = new SharedArray('txs', () => {
  try {
    const csv = open(CSV_PATH);
    const parsed = papaparse.parse(csv, { header: true, skipEmptyLines: true }).data;

    const mapped = parsed.map(r => {
      const amountMinor = toMinor(r.amount);
      return amountMinor == null ? null : {
        id: r.id,
        currency: String(r.currency || '').trim().toUpperCase(),
        amountMinor,
        sourceAccount: r.source_account,
        destinationAccount: r.destination_account,
      };
    });

    return mapped.filter(r => 
      r && r.currency && r.sourceAccount && Number.isInteger(r.amountMinor)
    );
  } catch (error) {
    console.error(`Error loading CSV: ${error}`);
    return [];
  }
});

export const options = {
  vus: Number(__ENV.VUS || 10),
  duration: __ENV.DURATION || '1m',
  thresholds: {
    'grpc_req_duration': ['p(95)<500'],
  }
};

export default () => {
  if (rows.length === 0) {
    console.error('No valid CSV rows found');
    return;
  }

  if (!client.connected) {
    try {
      client.connect(TARGET, { plaintext: true, timeout: '30s' });
    } catch (error) {
      console.error(`Connection failed: ${error}`);
      return;
    }
  }

  const r = rows[Math.floor(Math.random() * rows.length)];

  // FIXED: Use correct field names that match the proto definition
  const req = {
    id: r.id,  // This field exists in CSV but was missing from proto
    amount_minor: r.amountMinor,  // Fixed field name
    currency: r.currency === 'USD' ? 1 : 
             r.currency === 'IDR' ? 2 : 
             r.currency === 'SGD' ? 3 : 0,  // Convert string to enum value
    user_id: r.sourceAccount,  // Fixed field name
    destination_account: r.destinationAccount  // This field exists in CSV but was missing from proto
  };

  try {
    const resp = client.invoke('payments.v1.PaymentsService/CreatePayment', req, {
      timeout: '10s'
    });

    check(resp, {
      'status OK': (x) => x && x.status === grpc.StatusOK,
      'has response': (x) => x && x.message,
    });
  } catch (error) {
    console.error(`gRPC call failed: ${error}`);
  }

  sleep(0.05);
};