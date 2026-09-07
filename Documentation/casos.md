# Casos de uso y estrategia de pruebas

[Volver al resumen](Readme.md) · [Versión web](casos.html)

Catálogo de escenarios del discovery; consultar los informes de validación del repositorio para la evidencia ejecutada.

# 0. Casos de uso

## UC01 — Solicitar cotización

**Actor:** Cliente.

### Entrada

- Tenant.
- País.
- Cliente.
- Carrito.
- Medio de pago.

### Flujo principal

1. Validar solicitud.
2. Obtener contexto de país y tenant.
3. Obtener precios.
4. Obtener promociones vigentes.
5. Ejecutar el motor de precios.
6. Determinar obsequios.
7. Calcular base gravable.
8. Calcular impuestos.
9. Evaluar crédito.
10. Generar Quote Snapshot.
11. Retornar breakdown.

### Resultado

- `quoteId`.
- Líneas.
- Promociones.
- Obsequios.
- Impuestos.
- Total.
- Estado del crédito.

---

## UC02 — Confirmar pedido

**Actor:** Cliente.

### Precondiciones

- Existe una cotización.
- Existe `Idempotency-Key`.

### Flujo principal

1. Validar idempotencia.
2. Recuperar Quote Snapshot.
3. Validar disponibilidad de crédito.
4. Crear orden usando el snapshot.
5. Consumir crédito.
6. Persistir la orden.
7. Marcarla `CONFIRMED`.
8. Publicar `OrderConfirmed`.
9. Responder `201 Created`.

### Postcondición

```text
201 => Order persistida y consultable
```

---

## UC03 — Consultar pedido

**Actor:** Cliente.

### Entrada

`orderId`

### Resultado

Información de la orden confirmada.

---

## UC04 — Procesar pedido confirmado

**Actor:** Worker en Go.

### Trigger

Evento `OrderConfirmed` disponible en Pub/Sub.

### Flujo

1. Consumir evento.
2. Validar estructura.
3. Procesar de manera idempotente.
4. Enviar pedido hacia ERP.
5. Enviar Notificación Push.
6. Confirmar procesamiento del mensaje.

---

# 1. Casos de prueba

## 1.0. Casos funcionales

| ID | Caso | Resultado esperado |
|---|---|---|
| CF01 | Carrito sin promociones | Total = subtotal + impuestos |
| CF02 | Descuento por escala | Aplicar descuentos por tramo |
| CF03 | Combo válido | Aplicar precio/beneficio combo |
| CF04 | Obsequio válido | Mostrar SKU y cantidad gratis |
| CF05 | Pedido con impuestos | Impuesto calculado sobre base gravable |
| CF06 | Crédito suficiente | Pedido elegible |
| CF07 | Crédito insuficiente | Pedido no elegible |
| CF08 | Confirmar quote | Order.total = Quote.total |
| CF09 | Retry misma idempotency key | Misma orden |
| CF10 | Consultar orden confirmada | `200` y orden encontrada |

---

# 2. Casos de borde

## CB01 — Carrito vacío

**Entrada:** `items=[]`

**Esperado:** rechazo de validación.

---

## CB02 — Cantidad cero

```text
quantity = 0
```

**Esperado:** rechazo.

---

## CB03 — Cantidad negativa

**Esperado:** rechazo.

---

## CB04 — SKU inexistente

**Esperado:** error de dominio / producto no encontrado.

---

## CB05 — Promoción expirada

**Esperado:** no debe aplicarse.

---

## CB06 — Combo incompleto

Si falta cualquier cantidad mínima necesaria:

**Esperado:** combo no aplicado.

---

## CB07 — Exactamente en el límite de un tramo

Ejemplo:

```text
quantity = 10
```

Validar que la distribución entre tramos sea exacta.

---

## CB08 — Unidades excedentes al combo

Ejemplo:

```text
Combo requiere 2A + 1B

Carrito:
5A + 2B
```

Validar cuántos combos se aplican y qué unidades quedan disponibles para otras reglas.

---

## CB09 — Múltiples promociones candidatas

Validar:

- prioridad;
- compatibilidad;
- unidades consumidas;
- no doble descuento.

---

## CB10 — Impuesto con resultado decimal límite

Ejemplos:

```text
0.005
1.005
10.015
```

Validar política de rounding.

---

## CB11 — Crédito exactamente igual al total

```text
credit.available == order.total
```

**Esperado:** permitido.

---

## CB12 — Crédito menor por una unidad mínima

Ejemplo:

```text
total = 100.01
credit = 100.00
```

**Esperado:** rechazo.

---

## CB13 — Quote cambiado después de cotizar

Cambiar:

- precio;
- promoción;
- impuesto configurado;

antes de confirmar.

**Esperado:** se conserva el valor del snapshot.

---

## CB14 — Misma idempotency key con payload diferente

**Esperado:** `409 Conflict`.

---

## CB15 — Evento duplicado en Pub/Sub

**Esperado:** worker no debe producir efectos de negocio duplicados.

---

# 3. Casos límite

## CL01 — Cantidad máxima aceptada por línea

Definir un límite técnico configurable.

El motor debe:

- validar antes de procesar;
- evitar overflow;
- mantener tiempo de cálculo controlado.

---

## CL02 — Máximo de líneas por carrito

Ejemplo propuesto para la prueba:

```text
100 líneas
```

> Este valor es una **meta propuesta para el take-home**, no un requisito explícito del enunciado.

---

## CL03 — Máximo número de promociones aplicables

El motor debe poder procesar múltiples reglas sin generar doble aplicación sobre las mismas unidades.

---

## CL04 — Montos monetarios grandes

Probar cercanía al máximo soportado por el tipo decimal elegido.

---

## CL05 — Gran cantidad de reintentos

```text
N retries con misma Idempotency-Key
=> una sola orden
```

---

# 4. Casos de rendimiento

> El enunciado no entrega un SLA o throughput explícito. Los siguientes objetivos son **metas propuestas para la prueba**, enfocadas en detectar regresiones evidentes y demostrar criterio de ingeniería.

## CR01 — Pricing de carrito representativo

**Entrada propuesta:**

- 100 líneas.
- 20 promociones.
- Mezcla de combo, escala y obsequios.

**Objetivo:**

El cálculo debe ser determinístico y completar sin degradación significativa en ambiente local.

---

## CR02 — Cotizaciones concurrentes

Ejecutar múltiples solicitudes simultáneas de `POST /quotes`.

Validar:

- cero corrupción de datos;
- cero resultados no determinísticos;
- ausencia de race conditions.

---

## CR03 — Confirmaciones concurrentes sobre mismo crédito

Dos pedidos simultáneos compiten por un cupo que permite solo uno.

**Esperado:**

```text
1 confirmada
1 rechazada
```

Nunca ambas.

---

## CR04 — Reintentos concurrentes de la misma orden

Varias solicitudes con la misma `Idempotency-Key`.

**Esperado:**

```text
1 Order
1 consumo de crédito
```

---

## CR05 — Consumo sostenido de Pub/Sub

Publicar múltiples `OrderConfirmed`.

Validar:

- el worker continúa consumiendo;
- no bloquea la API;
- eventos fallidos pueden reintentarse;
- no existen efectos duplicados ante redelivery.

---

# 5. Casos Golden obligatorios

Los Golden Tests representan escenarios de negocio cuyo resultado debe permanecer estable ante refactors.

## GT01 — Descuento por escala en dos tramos

### Escenario

Un SKU alcanza dos tramos con descuento.

### Validar

- unidades asignadas correctamente;
- descuento de cada tramo;
- descuento total;
- total final.

---

## GT02 — Combo sin descuento suelto duplicado

### Escenario

Los productos activan un combo.

### Validar

- combo aplicado;
- origen visible;
- unidades consumidas por el combo;
- mismas unidades no aparecen también como descuento individual.

---

## GT03 — Obsequio

### Escenario

La compra cumple una condición que genera producto gratis.

### Validar

```text
gift.sku
gift.quantity
```

y presencia en el breakdown.

---

## GT04 — Combo con impuestos

### Validar

```text
combo
→ base gravable
→ impuesto
→ total
```

El impuesto debe formar parte del total.

---

## GT05 — Crédito excedido

### Escenario

```text
order.total > availableCredit
```

### Esperado

Pedido no elegible para crédito.

---

## GT06 — Regresión Quote / Order

### Escenario

1. Cotizar carrito.
2. Guardar quote.
3. Modificar precios/promociones vigentes.
4. Confirmar utilizando `quoteId`.

### Assert principal

```text
Quote.total == Order.total
```

---

# 6. Casos Golden adicionales recomendados

## GT07 — Idempotencia

```text
100 retries
=> 1 order
```

---

## GT08 — Crédito concurrente

Dos confirmaciones compiten por el mismo saldo.

```text
Solo una debe confirmar.
```

---

## GT09 — Rounding

Validar valores límite de decimal y tasas.

---

## GT10 — Multi-país

Mismo carrito con dos configuraciones de país.

Validar:

- moneda;
- escala;
- impuestos;
- rounding.

---

## GT11 — Evento OrderConfirmed

Confirmar pedido.

Validar que solo después del estado `CONFIRMED` exista publicación hacia Pub/Sub.

---

# 7. Mensajería: casos implementados de worker y DLQ

Evidencia: `backend/tests/integration/destination_test.go` y `node backend/scripts/dlq-smoke.mjs`.

| Escenario | Resultado esperado |
|---|---|
| Dos suscripciones | Un evento en orders-confirmed llega a ERP y PUSH; cada destino genera su recibo dummy. |
| Independencia | ERP falla y PUSH termina; también se verifica el caso inverso. |
| Retry durable | Cinco reintentos tras el inicial: 1/2/4/8/16 s. No ejecutar antes de nextAttempt ni reiniciar contador al recrear el consumidor. |
| Recuperación transitoria | Tras dos fallos el dummy termina; no se crea DLQ. |
| Presupuesto agotado | Sexto fallo: insertar DLQ y marcar destino terminal atómicamente antes del ACK. Un redelivery no vuelve a ejecutar la acción. |
| SMTP caído | El aviso permanece pendiente; al recuperar Mailpit se verifica el correo con identificador del evento y causa. |
| Mensaje malformado | Clave de retry estable, presupuesto acotado y error en DLQ. |
