import fs from 'node:fs';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { chromium } from '@playwright/test';

const output = fileURLToPath(new URL('../../Documentation/images/orden_calculo.svg', import.meta.url));
const ink = '#123B5D', teal = '#00A6A6';
let svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="1500" viewBox="0 0 1200 1500" role="img" aria-labelledby="title desc">
<title id="title">Seis razones para ordenar el cálculo del pedido</title>
<desc id="desc">Combo antes de escala para evitar beneficios duplicados; ajustes con origen; desglose explicable; impuestos después de descuentos; crédito sobre el total; snapshot completo para confirmar sin recalcular. Los ejemplos de unidades y montos son ilustrativos.</desc>
<defs><linearGradient id="bg" x2="0" y2="1"><stop stop-color="#edf8fc"/><stop offset="1" stop-color="#F5F7F8"/></linearGradient><marker id="arrow" viewBox="0 0 10 10" refX="9" refY="5" markerWidth="7" markerHeight="7" orient="auto"><path d="M0 0L10 5L0 10" fill="${teal}"/></marker></defs>
<rect width="1200" height="1500" fill="white"/><rect x="20" y="20" width="1160" height="1460" rx="22" fill="url(#bg)" stroke="#bfdde9"/><g font-family="Arial, sans-serif" fill="${ink}">`;
const esc = s => s.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;');
function text(x,y,s,size=20,bold=false,color=ink,anchor='start') { svg += `<text x="${x}" y="${y}" font-size="${size}" font-weight="${bold?700:400}" fill="${color}" text-anchor="${anchor}">${esc(s)}</text>`; }
function box(x,y,w,h,fill='#fff',stroke='#c9dde5',r=12) { svg+=`<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${r}" fill="${fill}" stroke="${stroke}"/>`; }
function arrow(d) { svg+=`<path d="${d}" fill="none" stroke="${teal}" stroke-width="3" marker-end="url(#arrow)"/>`; }
function panel(x,y,n,title,sub) { box(x,y,530,350);box(x+20,y+20,42,42,ink,ink,21);text(x+41,y+49,n,22,true,'white','middle');text(x+76,y+47,title,25,true);text(x+20,y+90,sub,19); }
function unit(x,y,fill) { box(x,y,27,32,fill,fill,5);svg+=`<path d="M${x+7} ${y+9}h13" stroke="white" stroke-width="3"/>`; }

text(55,66,'MARIPOSAMARKET  /  REGLAS DE PRECIO',17,true,teal);
text(55,115,'El orden del cálculo protege el pedido',39,true);
text(55,154,'Cada paso deja preparado el siguiente: un precio explicable y consistente.',23);

// A compact sequence gives the six visual explanations a shared context.
const steps=[['01','Combos'],['02','Escala'],['03','Obsequios'],['04','Impuestos'],['05','Total / crédito'],['06','Snapshot']];
steps.forEach(([n,label],i)=>{const x=55+i*184;box(x,190,166,62,i===5?'#fff3c9':'#e2f5f3',i===5?'#e6bf43':'#9fd8d3');text(x+83,213,n,14,true,teal,'middle');text(x+83,238,label,18,true,ink,'middle');if(i<5)arrow(`M${x+168} 221h13`);});

panel(55,282,'1','Evitar beneficios duplicados','El combo asigna unidades antes de la escala.');
text(80,413,'Ejemplo: 6 unidades elegibles',19,true);
box(80,430,282,103,'#e5f6f4','#9edbd5');box(380,430,178,103,'#eaf1ff','#b5caee');
for(let i=0;i<4;i++)unit(104+i*59,449,teal);
for(let i=0;i<2;i++)unit(416+i*65,449,'#527db5');
text(221,514,'4 asignadas al combo',18,true,ink,'middle');text(469,514,'2 restantes',18,true,ink,'middle');
text(80,565,'La escala evalúa las restantes, si cumplen',20);text(80,592,'su regla. Sin duplicar beneficios por defecto.',20);

panel(615,282,'2','Conservar el origen','Cada ajuste deja una ficha comercial legible.');
box(640,405,480,181,'#f5faff','#bcd7e8');
const fields=[['Promoción','COMBO-01'],['Tipo / origen','Combo / campaña'],['Unidades','4 unidades asignadas'],['Monto','Descuento aplicado']];
fields.forEach(([a,b],i)=>{text(660,438+i*39,a,19,true);text(835,438+i*39,b,19);});
text(640,612,'El beneficio puede explicarse y auditarse.',20);

panel(55,652,'3','Explicar el precio','El resumen conserva el desglose del cálculo.');
const receipt=[['Precio base','100'],['− Beneficios comerciales','− 10'],['+ Impuestos ilustrativos','+ 9'],['Total','99']];
box(80,773,300,183,'#f9fcfd');
receipt.forEach(([a,b],i)=>{text(96,804+i*41,a,17,i===3);text(362,804+i*41,b,20,true,ink,'end');});
box(398,801,160,113,'#fff5d2','#e7c96b');text(478,833,'OBSEQUIOS',16,true,ink,'middle');text(478,867,'+ 1 unidad',23,true,ink,'middle');text(478,896,'Cantidad visible',16,false,ink,'middle');
text(80,981,'Montos y obsequio ilustrativos; no son una tarifa.',17);

panel(615,652,'4','Gravar la base correcta','Primero descuentos, después impuestos.');
box(640,782,138,78,'#eaf1ff');box(806,782,138,78,'#e5f6f4');box(972,782,148,78,'#fff5d2','#e7c96b');
text(709,812,'Precio base',17,true,ink,'middle');text(709,844,'100',26,true,ink,'middle');text(792,832,'−',27,true);
text(875,812,'Descuentos',17,true,ink,'middle');text(875,844,'10',26,true,ink,'middle');text(951,832,'=',25,true);
text(1046,812,'Base gravable',17,true,ink,'middle');text(1046,844,'90',26,true,ink,'middle');
arrow('M1046 865v30');text(640,925,'Impuesto = 90 × tasa configurada',23,true);text(640,964,'La base refleja el precio después del beneficio.',19);

panel(55,1022,'5','Evaluar el cupo real','En una compra a crédito se compara el total.');
text(80,1161,'Cupo disponible',19,true);box(80,1177,470,36,'#e7eef1','#e7eef1',8);box(80,1177,388,36,teal,teal,8);text(535,1202,'120',19,true,ink,'end');
text(80,1243,'Pedido: 99 de 120',23,true);text(80,1280,'Incluye impuestos y descuentos definitivos.',19);text(80,1318,'Al confirmar se valida y reserva el cupo.',20);text(80,1349,'Ejemplo ilustrativo de un pedido con crédito.',17);

panel(615,1022,'6','Respetar Quote = Order','El snapshot se guarda al terminar el cálculo.');
box(640,1149,207,117,'#eaf1ff','#b5caee');box(913,1149,207,117,'#e5f6f4','#9edbd5');
text(743,1183,'QUOTE',21,true,ink,'middle');text(743,1216,'Snapshot completo',18,true,ink,'middle');text(743,1246,'Total: 99',22,true,ink,'middle');
arrow('M853 1207h52');
text(1016,1183,'ORDER',21,true,ink,'middle');text(1016,1216,'Mismo desglose',18,true,ink,'middle');text(1016,1246,'Total: 99',22,true,ink,'middle');
text(640,1310,'Confirmar sin recalcular precios ni promociones.',19);text(640,1344,'Una cotización válida conserva lo mostrado.',19);

box(55,1400,1090,52,ink,ink);text(600,1433,'Beneficios claros  →  total definitivo  →  pedido consistente',23,true,'white','middle');
svg+='</g></svg>\n';
fs.writeFileSync(output,svg);
const browser=await chromium.launch({channel:'chrome',headless:true});
try {
 const page=await browser.newPage({viewport:{width:1200,height:1500},deviceScaleFactor:1});
 await page.goto(pathToFileURL(output).href);
 const overflow=await page.locator('text').evaluateAll(nodes=>nodes.filter(n=>{const b=n.getBBox();return b.x<0||b.x+b.width>1200;}).map(n=>n.textContent));
 if(overflow.length)throw new Error(`Text outside infographic: ${overflow.join(', ')}`);
 await page.screenshot({path:output.replace('.svg','.png')});
} finally { await browser.close(); }
console.log('Pricing infographic SVG and PNG generated.');
