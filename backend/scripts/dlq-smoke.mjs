// Local Docker demo only. Creates three CASH orders and targeted dummy failures.
// Restores worker/mailpit in finally; no data or mail is deleted.
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import assert from 'node:assert/strict';
const compose=(...args)=>execFileSync('docker',['compose',...args],{encoding:'utf8',timeout:60000});
const mongo=code=>JSON.parse(compose('exec','-T','mongo','mongosh','--quiet','marketplace','--eval',`JSON.stringify(${code})`).trim());
const headers={'Content-Type':'application/json','X-Tenant-ID':'tenant-demo','X-Country':'PE','X-Customer-ID':'CUSTOMER-001'};
async function request(path,body,extra={}) {
  const r=await fetch('http://localhost:8080'+path,{method:'POST',headers:{...headers,...extra},body:JSON.stringify(body),signal:AbortSignal.timeout(15000)});
  const data=await r.json();assert.equal(r.status,201,JSON.stringify(data));return data;
}
async function create(kind,failures) {
  const q=await request('/quotes',{tenantId:'tenant-demo',country:'PE',customerId:'CUSTOMER-001',paymentMethod:'CASH',items:[{sku:'SKU-001',quantity:12}]});
  const o=await request('/orders',{quoteId:q.quoteId,customerId:'CUSTOMER-001'},{'Idempotency-Key':randomUUID()});
  const event=mongo(`db.outbox_events.findOne({'order.orderId':${JSON.stringify(o.orderId)}})`);
  mongo(`db.dummy_failures.insertOne(${JSON.stringify({_id:event.eventId+':'+kind,remaining:failures})})`);
  return event.eventId;
}
async function until(check,label) {
  const end=Date.now()+120000;
  while(Date.now()<end) {if(await check())return;await new Promise(r=>setTimeout(r,1000));}
  throw Error('Timed out: '+label);
}
try {
  compose('stop','worker','mailpit');
  const erp=await create('erp',6),push=await create('push',6),transient=await create('erp',2);
  const deadIds=[erp+':erp',push+':push'];
  compose('start','worker');
  await until(()=>mongo(`db.dead_letters.countDocuments({_id:{$in:${JSON.stringify(deadIds)}}})`)===2,'six failed attempts per destination');
  for(const [id,failed,successful] of [[erp,'erp','push'],[push,'push','erp']]) {
    const state=mongo(`db.processed_events.findOne({eventId:${JSON.stringify(id)}})`);
    assert.equal(state.effects[failed].failures,6);
    assert.equal(state.effects[failed].dead,true);
    assert.equal(state.effects[successful].done,true);
    assert.equal(mongo(`db.mock_integration_receipts.countDocuments({_id:${JSON.stringify(id+':'+successful)}})`),1);
    assert.equal(mongo(`db.mock_integration_receipts.countDocuments({_id:${JSON.stringify(id+':'+failed)}})`),0);
  }
  const recovered=mongo(`db.processed_events.findOne({eventId:${JSON.stringify(transient)}})`);
  assert.equal(recovered.effects.erp.failures,2);
  assert.equal(recovered.effects.erp.done,true);assert.equal(recovered.effects.push.done,true);
  assert.equal(mongo(`db.dead_letters.countDocuments({_id:{$in:${JSON.stringify(deadIds)}},emailedAt:{$exists:true}})`),0);
  await until(()=>mongo(`db.dead_letters.countDocuments({_id:{$in:${JSON.stringify(deadIds)}},mailError:{$exists:true,$ne:''}})`)>0,'SMTP failure retained');
  compose('start','mailpit');
  await until(()=>mongo(`db.dead_letters.countDocuments({_id:{$in:${JSON.stringify(deadIds)}},emailedAt:{$exists:true}})`)===2,'mail retry');
  const list=await(await fetch('http://localhost:8025/api/v1/messages?limit=100')).json();
  const mails=await Promise.all(list.messages.map(async m=>(await(await fetch('http://localhost:8025/api/v1/message/'+m.ID)).json()).Text));
  for(const id of deadIds)assert.ok(mails.some(text=>text.includes(id)),'missing email for '+id);
  console.log('PASS: independent ERP/PUSH subscriptions, transient recovery, six failures -> durable DLQ, SMTP outage -> email retry; inbox http://localhost:8025');
} finally {compose('start','mailpit','worker');}
