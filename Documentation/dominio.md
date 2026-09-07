# Dominio y reglas de negocio

[Volver al resumen](Readme.md) · [Versión web](dominio.html)

# 0. Dominio de negocio

## 0.0. Agregados y entidades principales

### Customer

Representa al comprador B2B.

```text
Customer
- id
- tenantId
- country
- creditAccount
```

### Product

```text
Product
- sku
- description
- family
- unitPrice
- taxCategory
```

### Cart

```text
Cart
- customerId
- tenantId
- country
- items[]
```

### CartItem

```text
CartItem
- sku
- quantity
```

### Promotion

Representa una palanca comercial.

```text
Promotion
- id
- type
- priority
- eligibility
- compatibility
- validFrom
- validTo
```

Tipos iniciales:

```text
SCALE
COMBO
GIFT
```

### Quote

Representa la cotización calculada y aceptable por el cliente.

```text
Quote
- quoteId
- tenantId
- country
- customerId
- currency
- lines[]
- adjustments[]
- gifts[]
- grossSubtotal
- taxableBase
- taxTotal
- total
- creditEvaluation
- createdAt
```

### Order

```text
Order
- orderId
- orderNumber
- quoteId
- customerId
- tenantId
- status
- total
- createdAt
```

### CreditAccount

```text
CreditAccount
- customerId
- availableAmount
- currency
```

### OrderConfirmedEvent

```text
OrderConfirmedEvent
- eventId
- orderId
- orderNumber
- customerId
- tenantId
- total
- currency
- occurredAt
```

---

# 1. Reglas de negocio

## RN01 — Dinero exacto

No se debe utilizar `float32` ni `float64` para representar dinero.

El tipo monetario debe encapsular:

```text
Money
- amount
- currency
- scale
```

La política de redondeo debe ser explícita y configurable por país/moneda.

---

## RN02 — Configuración multi-país

Los siguientes valores no deben quedar hardcodeados en el motor:

- Moneda.
- Escala decimal.
- Regla de redondeo.
- Tasas tributarias.
- Configuración comercial específica del país.

---

## RN03 — Cotización determinística

Para el mismo:

```text
Carrito + Configuración + Reglas + Precios
```

el resultado debe ser el mismo.

---

## RN04 — Quote Snapshot

Una vez generada la cotización, el resultado económico utilizado para confirmar el pedido debe mantenerse inmutable.

```text
Quote.total == Order.total
```

La confirmación no debe recalcular el pedido usando nuevos precios o promociones.

---

## RN05 — Descuento por escala

Las escalas son acumulativas por tramos.

Ejemplo:

```text
1 - 5 unidades    : 0%
6 - 10 unidades   : 5%
11+ unidades      : 10%
```

Para 12 unidades:

```text
1 - 5   => 0%
6 - 10  => 5%
11 - 12 => 10%
```

---

## RN06 — Combo

Los productos y cantidades usados para activar un combo deben quedar asociados al combo.

Por defecto, esas mismas unidades no deben recibir adicionalmente un descuento suelto por escala.

---

## RN07 — Obsequio

Cuando se cumpla la condición comercial:

- El SKU regalado debe aparecer explícitamente.
- Debe indicarse la cantidad.
- El valor comercial puede ser cero para el cliente.
- Su tratamiento tributario debe depender de la configuración del país/regla.

---

## RN08 — Impuestos

Los impuestos se calculan sobre la base gravable resultante **después de descuentos comerciales**.

```text
Subtotal bruto
- descuentos comerciales
= base gravable
```

Luego:

```text
base gravable × tasa = impuesto
```

---

## RN09 — Crédito

Si el medio de pago es crédito:

```text
Order.total <= Credit.available
```

La validación y consumo deben evitar sobreconsumo ante solicitudes concurrentes.

---

## RN10 — Idempotencia

`POST /orders` debe recibir una `Idempotency-Key`.

```text
Misma key + mismo request
=> misma orden
```

```text
Misma key + request diferente
=> conflicto
```

---

## RN11 — Confirmación

La API solo debe devolver `201 Created` cuando la orden haya sido persistida correctamente.

---

## RN12 — Evento de pedido confirmado

Solo se publica `OrderConfirmed` cuando la orden alcanzó estado `CONFIRMED`.

El evento no debe representar un intento de confirmación sino un hecho ya ocurrido.

---

# 2. Entrega, reintentos y DLQ

Entrega por destino: orders-confirmed alimenta dos suscripciones independientes (ERP y PUSH). Cada acción dummy tiene ACK propio y cinco reintentos exponenciales tras el intento inicial. Al agotar intentos se persiste la DLQ en Mongo antes del ACK; el aviso SMTP se reintenta sin perder el registro.
