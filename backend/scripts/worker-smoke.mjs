// Run from the repository root against the local Docker demo (mock ERP/PUSH).
// Stops worker and broker temporarily; always starts them again. Keeps demo orders.
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import assert from 'node:assert/strict';

const compose = (...args) => execFileSync('docker', ['compose', ...args], { encoding: 'utf8', timeout: 60000 });
const mongo = code => JSON.parse(compose('exec', '-T', 'mongo', 'mongosh', '--quiet', 'marketplace', '--eval', `JSON.stringify(${code})`).trim());
const headers = { 'Content-Type': 'application/json', 'X-Tenant-ID': 'tenant-demo', 'X-Country': 'PE', 'X-Customer-ID': 'CUSTOMER-001' };
async function request(path, status, body, extra = {}) {
  const res = await fetch('http://localhost:8080' + path, {
    method: body ? 'POST' : 'GET', headers: { ...headers, ...extra },
    ...(body ? { body: JSON.stringify(body) } : {}), signal: AbortSignal.timeout(15000),
  });
  const data = await res.json();
  assert.equal(res.status, status, JSON.stringify(data));
  return data;
}
async function order() {
  const q = await request('/quotes', 201, { tenantId: 'tenant-demo', country: 'PE', customerId: 'CUSTOMER-001', paymentMethod: 'CASH', items: [{ sku: 'SKU-001', quantity: 12 }] });
  const o = await request('/orders', 201, { quoteId: q.quoteId, customerId: 'CUSTOMER-001' }, { 'Idempotency-Key': randomUUID() });
  assert.deepEqual(o.total, q.total);
  return o;
}
try {
  compose('stop', 'worker', 'pubsub-emulator');
  assert.equal((await request('/ready', 200)).status, 'ready');
  const pending = await order();
  const lost = await order();
  const filter = JSON.stringify({ 'order.orderId': { $in: [pending.orderId, lost.orderId] } });
  assert.equal(mongo(`db.outbox_events.countDocuments({...${filter}, publishedAt: {$exists:false}})`), 2);
  // Simulate a previous broker acknowledgement whose message was lost.
  mongo(`db.outbox_events.updateOne({'order.orderId':${JSON.stringify(lost.orderId)}}, {$set:{publishedAt:new Date(Date.now()-120000)}})`);
  const events = mongo(`db.outbox_events.find(${filter}).toArray()`);
  const ids = events.map(e => e.eventId);
  compose('start', 'pubsub-emulator', 'worker');
  const deadline = Date.now() + 90000;
  let completed = 0;
  while (Date.now() < deadline) {
    completed = mongo(`db.processed_events.countDocuments({eventId:{$in:${JSON.stringify(ids)}},'effects.erp.done':true,'effects.push.done':true})`);
    if (completed === 2) break;
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  assert.equal(completed, 2, 'Worker must publish pending/lost events and complete both destinations');
  const receipts = mongo(`db.mock_integration_receipts.find({'event.eventId':{$in:${JSON.stringify(ids)}}}).toArray()`);
  for (const id of ids) assert.deepEqual(receipts.filter(r => r.event.eventId === id).map(r => r.kind).sort(), ['erp', 'push']);
  assert.equal(mongo(`db.outbox_events.countDocuments({...${filter},publishedAt:{$exists:true}})`), 2);
  console.log('PASS: API independent of broker/worker; pending and lost publications recovered; ERP/PUSH completed once per event.');
} finally {
  compose('start', 'pubsub-emulator', 'worker');
}
