import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from '@playwright/test';

const root = fileURLToPath(new URL('../../', import.meta.url));
const output = path.join(root, 'Documentation/images/arquitectura.svg');
const ink = '#101b46';
let svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1800" height="1220" viewBox="0 0 1800 1220" role="img" aria-labelledby="title desc">
<title id="title">Arquitectura de componentes — MariposaMarket</title>
<desc id="desc">Clientes web y Postman acceden a Flutter/nginx o a la API Go. Docker Compose contiene frontend, API, MongoDB, emulador Pub/Sub, worker y Mailpit. La API persiste pedido, crédito, idempotencia y outbox atómicamente. El worker publica pendientes en orders-confirmed, consume dos suscripciones independientes ERP y PUSH, ejecuta acciones dummy y guarda progreso. Cinco reintentos exponenciales tras el inicial; después del sexto fallo guarda DLQ en Mongo antes del ACK y envía correo local.</desc>
<defs>
<linearGradient id="compose" x2="0" y2="1"><stop stop-color="#f4fbff"/><stop offset="1" stop-color="#f8fcff"/></linearGradient>
<linearGradient id="green" x2="0" y2="1"><stop stop-color="#e6faef"/><stop offset="1" stop-color="#f8fdfa"/></linearGradient>
<linearGradient id="blue" x2="0" y2="1"><stop stop-color="#e2f3ff"/><stop offset="1" stop-color="#f4faff"/></linearGradient>
<linearGradient id="purple" x2="0" y2="1"><stop stop-color="#eee6ff"/><stop offset="1" stop-color="#faf7ff"/></linearGradient>
<marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto-start-reverse"><path d="M0 0L10 5L0 10" fill="#203d68"/></marker>
</defs><rect width="1800" height="1220" fill="white"/>
<g font-family="Arial, sans-serif" fill="${ink}">`;
const esc = s => s.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;');
function text(x,y,s,size=20,bold=false,anchor='start',color=ink) {
  svg += `<text x="${x}" y="${y}" font-size="${size}"${bold?' font-weight="700"':''} text-anchor="${anchor}" fill="${color}">${esc(s)}</text>`;
}
function box(x,y,w,h,fill,stroke='#8cbbdf',radius=12) { svg+=`<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${radius}" fill="${fill}" stroke="${stroke}"/>`; }
function logo(name,x,y,w=45,h=42) {
 const data=fs.readFileSync(path.join(root,'Documentation/images/logos',name+'.svg')).toString('base64');
 svg+=`<image x="${x}" y="${y}" width="${w}" height="${h}" preserveAspectRatio="xMidYMid meet" href="data:image/svg+xml;base64,${data}"/>`;
}
function line(d,dashed=false,both=false) { svg+=`<path d="${d}" fill="none" stroke="#203d68" stroke-width="2.3" ${dashed?'stroke-dasharray="6 5" ':''}marker-end="url(#arrow)"${both?' marker-start="url(#arrow)"':''}/>`; }
function icon(kind,x,y,color='#087ee5',scale=1) {
 const shapes={
  doc:'<path d="M6 2h19l9 9v31H6z"/><path d="M25 2v11h9M12 20h16M12 27h16M12 34h12"/>',
  cart:'<path d="M2 5h7l5 24h23l5-17H11"/><circle cx="18" cy="38" r="3"/><circle cx="34" cy="38" r="3"/>',
  shield:'<path d="M22 2L5 9v12c0 11 17 21 17 21s17-10 17-21V9zM13 21l6 6 12-13"/>',
  mail:'<rect x="3" y="8" width="38" height="28" rx="3"/><path d="M3 10l19 15 19-15"/>',
  db:'<ellipse cx="22" cy="7" rx="17" ry="6"/><path d="M5 7v30c0 8 34 8 34 0V7M5 18c0 8 34 8 34 0M5 28c0 8 34 8 34 0"/>',
  net:'<circle cx="22" cy="22" r="5"/><circle cx="22" cy="3" r="2"/><circle cx="3" cy="22" r="2"/><circle cx="41" cy="22" r="2"/><circle cx="22" cy="41" r="2"/><path d="M22 5v12m0 10v12M5 22h12m10 0h12"/>',
  cube:'<path d="M22 2L3 12v22l19 10 19-10V12zM3 12l19 11 19-11M22 23v21"/>',
  web:'<rect x="3" y="4" width="38" height="28" rx="2"/><path d="M22 32v7M9 40h26"/>',
  phone:'<rect x="11" y="1" width="24" height="42" rx="4"/><path d="M19 6h8M20 37h6"/>',
  bell:'<path d="M8 31h28l-4-7V14c0-13-20-13-20 0v10zM17 37c0 7 10 7 10 0"/>',
  tag:'<path d="M3 23L23 3h17v17L20 40z"/><circle cx="32" cy="11" r="3"/>',
 };
 svg+=`<g transform="translate(${x} ${y}) scale(${scale})" fill="none" stroke="${color}" stroke-width="2.8" stroke-linecap="round" stroke-linejoin="round">${shapes[kind]}</g>`;
}

text(32,58,'Arquitectura de Componentes',46,true);
text(32,100,'MariposaMarket · API y worker en Go + MongoDB + Pub/Sub + Flutter',27);
box(315,135,1460,925,'url(#compose)','#55b8fa');
logo('docker',339,148,60,44); text(412,180,'Docker Compose',26,true);text(650,180,'(entorno local / desarrollo)',20);

// External clients: mobile is responsive web, not an unimplemented native app.
box(25,315,240,420,'url(#blue)','#229dff');text(44,351,'Clientes /',23,true);text(44,379,'Consumidores',23,true);
box(37,402,216,316,'#f4faff','#f4faff');
icon('web',53,431,'#075895');text(114,449,'Navegador',19,true);text(114,475,'Web',17);
icon('phone',53,523,'#075895');text(114,541,'Web móvil',19,true);text(114,567,'Adaptable',17);
logo('postman',51,617,47,43);text(114,635,'Postman',19,true);text(114,661,'Pruebas REST',17);

// Frontend and API have separate Compose services.
box(355,211,550,88,'#f0faff','#54b8ee');logo('flutter',372,232);logo('nginx',425,237,64,34);
text(507,243,'Frontend · Flutter Web + nginx',23,true);text(507,276,'UI adaptable · proxy /api → api:8080',18);
box(355,328,550,422,'url(#green)','#26ad5b');logo('go',374,341,45,42);text(435,370,'Aplicación API · Go',26,true);
box(375,393,510,70,'url(#blue)','#5daff0');icon('net',395,406);text(465,423,'Transporte HTTP',22,true);text(465,451,'Router, contexto y validación del contrato',18);
const modules=[['doc','Quote','Snapshot',375,476],['cart','Order','Confirmación',548,476],['tag','Pricing Engine','Promociones',721,476],['db','Credit','Cupo disponible',375,560],['shield','Idempotency','Clave + hash',548,560],['mail','Eventos','Por publicar',721,560]];
for(const [i,title,sub,x,y] of modules){box(x,y,164,73,'#f8fcff','#8cb7d2');icon(i,x+12,y+18,'#087ee5',.75);text(x+53,y+28,title,title.length>12?15:16,true);text(x+53,y+53,sub,13);}
box(375,647,510,42,'#eae2ff','#ae85fa');icon('cube',393,653,'#6c42c5',.65);text(442,674,'Dominio · entidades y reglas de negocio',19,true);
box(375,701,510,36,'#edf0f3','#9dabb8');icon('db',393,704,'#466079',.6);text(442,725,'Puertos y repositorios · adaptador MongoDB',18,true);

line('M265 452H345',false,true);text(305,410,'HTTP / JSON',14,false,'middle');text(305,432,'REST API',14,false,'middle');
line('M145 315V255H345',true);text(208,243,'UI',14);
line('M630 299V318');
line('M630 750V825');text(651,783,'Driver MongoDB',17);text(651,810,'Commit antes del 201',17,true);

// Mongo is also the durable retry/DLQ store.
box(355,835,550,200,'url(#green)','#26ad5b');logo('mongodb',377,848,38,41);text(432,877,'MongoDB · replica set',24,true);
box(373,900,514,119,'#f9fdfb','#b3e0c4');
const cols=[['products / customers','promotions / countries','credits'],['quotes / orders','idempotency','outbox_events'],['processed_events','dead_letters (DLQ)','recibos dummy']];
cols.forEach((items,c)=>items.forEach((s,r)=>{icon('db',385+c*170,910+r*34,'#52749a',.48);text(412+c*170,926+r*34,s,13.5);}));

// One business topic, independent subscriptions.
box(957,305,313,432,'url(#blue)','#299fff');logo('googlecloud',976,325,43,40);text(1031,346,'Google Pub/Sub',22,true);text(1031,374,'Emulador Docker',17);
box(975,393,277,78,'#f8fcff','#79bdf5');icon('net',990,414,'#0886f0',.7);text(1034,420,'Tópico',19,true);text(1113,451,'orders-confirmed',18,true,'middle');
box(975,492,277,88,'#f8fcff','#79bdf5');icon('net',990,514,'#0886f0',.7);text(1034,520,'Suscripción ERP',19,true);text(1113,554,'orders-confirmed-erp',17,true,'middle');
box(975,613,277,88,'#f8fcff','#79bdf5');icon('net',990,635,'#0886f0',.7);text(1034,641,'Suscripción PUSH',19,true);text(1113,675,'orders-confirmed-push',17,true,'middle');

// Four concurrent loops in one independent worker application.
box(1325,231,425,615,'url(#purple)','#ad7cff');logo('go',1344,250,44,39);text(1402,279,'Aplicación Worker · Go',25,true);
box(1345,310,385,98,'#faf7ff','#b79bea');icon('mail',1361,334,'#7843c3',.8);text(1411,339,'Publicador de confirmaciones',20,true);text(1411,368,'Publica en Pub/Sub y recupera pendientes',16);text(1411,392,'Mismo eventId · entrega al menos una vez',15);
box(1345,442,385,102,'#f8faff','#a6b4e5');icon('doc',1361,466,'#42668e',.8);text(1411,471,'Consumidor ERP',21,true);text(1411,501,'Acción dummy → progreso → ACK ERP',16);text(1411,525,'Fallo: sin ACK y reintento programado',15);
box(1345,565,385,102,'#f7fcfa','#9fcfb7');icon('bell',1361,589,'#27778b',.8);text(1411,594,'Consumidor PUSH',21,true);text(1411,624,'Acción dummy → progreso → ACK PUSH',16);text(1411,648,'Independiente del resultado de ERP',15);
box(1345,691,385,131,'#fff8f0','#deae77');icon('mail',1361,716,'#ad7516',.8);text(1411,721,'DLQ y alertas',21,true);text(1411,752,'6.º fallo: persistir DLQ antes del ACK',16);text(1411,779,'Cola durable dead_letters en Mongo',16);text(1411,807,'Bucle SMTP · reintentar correo fallido',15);

// Explicit routing: publisher into topic; subscriptions into consumers.
box(957,211,313,64,'#e3f7f5','#00a6a6');text(1113,236,'Worker → Pub/Sub',21,true,'middle');text(1113,261,'Publica OrderConfirmed',19,true,'middle');
svg+='<path d="M1345 357H1290V285H1113V305" fill="none" stroke="#00a6a6" stroke-width="4" marker-end="url(#arrow)"/>';
line('M1252 536H1285V493H1335');
line('M1252 657H1298V617H1335');
line('M905 876H931V772H1311V385H1335',true);text(1083,760,'Lee eventos por publicar',16,false,'middle');
line('M1345 793H1285V965H915',true);text(1092,949,'Progreso, retries y DLQ',17,false,'middle');

box(1420,920,310,112,'#fff7e4','#e2af4a');icon('mail',1438,943,'#a77215',.85);text(1494,952,'Mailpit · SMTP local',21,true);text(1494,982,'Bandeja de prueba',18);text(1494,1010,'localhost:8025',17);
line('M1540 822V910');text(1558,876,'Correo del problema',17);text(1558,899,'Retry SMTP: 30 s',16);

// Match the reference's service inventory footer without hiding extra services.
box(25,1080,1750,116,'#fafcff','#bacde1');logo('docker',46,1114,63,47);text(126,1132,'Servicios Docker Compose',22,true);text(126,1162,'1 intento + 5 retries: 1 / 2 / 4 / 8 / 16 s',17);
const services=[['frontend','Flutter + nginx','#e8f7ff'],['api','Go','#e4f8eb'],['mongo','Replica set','#e4f8eb'],['pubsub-emulator','Pub/Sub','#e8f2ff'],['worker','Go','#eee5ff'],['mailpit','SMTP + bandeja','#fff6df'],['seed','Carga inicial','#f0f2f6']];
services.forEach(([a,b,c],i)=>{const x=540+i*171;box(x,1100,156,75,c,'#b0c3d8');text(x+78,1129,a,a.length>12?15:19,true,'middle');text(x+78,1157,b,16,false,'middle');});
svg+='</g></svg>';
fs.writeFileSync(output,svg+'\n');
const browser=await chromium.launch({channel:'chrome',headless:true});
try {
 const page=await browser.newPage({viewport:{width:1800,height:1220},deviceScaleFactor:1});
 await page.goto(pathToFileURL(output).href);
 await page.screenshot({path:output.replace('.svg','.png')});
} finally { await browser.close(); }
console.log('Architecture SVG and PNG generated.');
