# Prueba Técnica Tech Lead — Discovery y Diseño de Solución

[MariposaMarket · ADR de una página](adr.html) · [Descargar PDF](ADR.pdf)

## Marketplace B2B Multi-tenant — Motor de Precios y Confirmación de Pedidos

> **Objetivo del documento**
> Consolidar el Discovery funcional y técnico de la solución propuesta para la prueba técnica: problema, flujo funcional, journey del cliente, dominio y reglas de negocio, casos de uso, estrategia de pruebas, casos golden, orden de aplicación de palancas comerciales y stack tecnológico.

---

# 0. Descripción del problema

La plataforma corresponde a un **marketplace B2B multi-tenant** donde tenderos compran productos a distribuidores de bebidas.

El journey principal de compra es:

```text
Catálogo → Carrito → Checkout / Cotización → Confirmación → Historial
```

Sobre un mismo pedido pueden coexistir distintas **palancas comerciales**:

- Descuento por escala.
- Combo.
- Obsequio.
- Impuestos.
- Cupo de crédito.

La solución debe resolver dos problemas principales:

## 0.0. Motor de cálculo del pedido

Dado un carrito y un conjunto de reglas comerciales vigentes, el sistema debe calcular:

- Subtotal bruto.
- Descuentos aplicados.
- Origen de cada descuento.
- Combos.
- Obsequios y sus cantidades.
- Base gravable.
- Impuestos.
- Total a pagar.
- Evaluación del cupo de crédito.

El resultado debe ser **explicable**, es decir, no basta con entregar un total: el cliente debe poder entender cómo se obtuvo.

## 0.1. Confirmación confiable del pedido

La confirmación debe garantizar que:

1. El total confirmado sea exactamente el mismo que el total cotizado.
2. Los reintentos no creen pedidos duplicados.
3. El cupo de crédito no se consuma dos veces.
4. Si se responde `201 Created`, la orden ya exista y pueda consultarse.
5. Luego de confirmar el pedido, se publique un evento para integraciones externas.
6. ERP y Notificaciones Push se procesen de forma desacoplada respecto de la confirmación del pedido.

## 0.2. Invariantes principales

```text
I1. Quote.total == Order.total

I2. HTTP 201 => Order existe y es consultable

I3. Misma Idempotency-Key + mismo request => misma Order

I4. Una intención lógica de compra => un único consumo de crédito

I5. El dinero nunca usa aritmética floating point

I6. Cada descuento o beneficio debe ser trazable a su regla de origen

I7. País, moneda, escala decimal e impuestos no deben estar hardcodeados

I8. Pedido confirmado => evento de pedido confirmado publicable hacia Pub/Sub
```

---

# 1. Diagrama funcional

[![Diagrama funcional del marketplace B2B](images/diagrama_funcional.png)](images/diagrama_funcional.png)

## 1.0. Lectura funcional del flujo

1. El cliente construye su carrito.
2. Solicita una cotización.
3. El sistema calcula beneficios, impuestos y crédito.
4. El cliente revisa el resumen antes de comprar.
5. Al confirmar, el sistema valida idempotencia y crédito.
6. La orden se persiste antes de entregar una respuesta exitosa.
7. Una vez confirmada, la API publica un evento.
8. Un worker en Go consume el evento desde Pub/Sub.
9. El worker integra con ERP y Notificaciones Push.
10. La orden confirmada queda disponible en historial.

---

# 2. Journey del cliente

![Journey del cliente](images/journey_de_cliente.png)

## 2.0. Momentos clave del Journey

| Momento | Expectativa del cliente | Riesgo | Respuesta de diseño |
|---|---|---|---|
| Cotización | Entender cuánto pagará | Total opaco | Breakdown completo |
| Promociones | Entender el beneficio recibido | Descuento inexplicable | Origen de cada ajuste |
| Confirmación | Pagar exactamente lo cotizado | Precio diferente al confirmar | Quote Snapshot |
| Reintento | No duplicar la compra | Doble tap / mala red | Idempotency-Key |
| Crédito | Saber si puede comprar | Sobreconsumo concurrente | Validación y consumo atómico |
| Confirmación | Tener certeza de compra | `201` sin orden persistida | Responder después de persistir |
| Historial | Encontrar su pedido | Pedido “desaparece” | Orden consultable inmediatamente |

---

# 3. Dominio y reglas de negocio

[Ver entidades y reglas de negocio](dominio.md).

# 4. Casos de uso y pruebas

[Ver casos de uso, pruebas, bordes, límites, rendimiento y golden tests](casos.md).

# 5. Orden de aplicación de palancas

## 5.0. Decisión

El orden propuesto es:

```text
1. Resolver contexto Tenant / País
          ↓
2. Resolver precios base
          ↓
3. Detectar y asignar Combos
          ↓
4. Aplicar descuentos por Escala
   sobre unidades elegibles restantes
          ↓
5. Determinar Obsequios
          ↓
6. Calcular Base Gravable
          ↓
7. Calcular Impuestos
          ↓
8. Calcular Totales
          ↓
9. Evaluar Cupo de Crédito
          ↓
10. Generar Quote Snapshot
```

---

# 6. Justificación del orden de palancas

La decisión se basa en las perspectivas levantadas durante el Discovery con dos perfiles de negocio.

## 6.0. Perspectiva de Gerencia de Marketing Digital

La mirada de Marketing prioriza:

- que el beneficio sea visible;
- que las promociones sean explicables;
- que el cliente confíe en el precio;
- que las campañas puedan coexistir bajo reglas claras;
- que se conozca qué promociones son acumulables y cuáles excluyentes.

Esto implica que el motor debe mantener trazabilidad de:

```text
Qué regla se aplicó
Sobre qué unidades
Qué beneficio produjo
Qué reglas quedaron excluidas
```

### Efecto en el orden

**Combo antes que descuento por escala**.

Un combo consume un conjunto concreto de productos y cantidades. Si primero se aplica un descuento individual y luego se arma un combo con las mismas unidades, el cliente podría recibir un beneficio duplicado no intencional y el breakdown sería difícil de explicar.

Por eso:

```text
Combo
→ reserva/asigna unidades
→ Scale Discount solo sobre unidades restantes elegibles
```

---

## 6.1. Perspectiva del Product Owner B2B

La mirada de Producto prioriza:

1. Un motor mínimo pero confiable.
2. Journey completo antes que sofisticación innecesaria.
3. Que el precio mostrado se respete al confirmar.
4. Que el crédito represente correctamente la capacidad real de compra.

### Efecto en el orden

**Impuestos después de descuentos** porque la base gravable debe reflejar el precio comercial resultante.

**Crédito después del total** porque solo puede evaluarse cuando ya se conoce el valor definitivo del pedido.

**Quote Snapshot al final** porque debe persistir exactamente el resultado completo que el cliente visualizó.

---

# 7. ¿Por qué este orden es correcto?

![Infografía: seis razones del orden de cálculo, desde evitar beneficios duplicados hasta confirmar el pedido sin recalcular.](images/orden_calculo.svg)

[Abrir en tamaño completo](images/orden_calculo.svg) · [Descargar PNG](images/orden_calculo.png)

Los montos y las unidades de la infografía son ilustrativos. Los impuestos usan la tasa configurada; el cupo se evalúa en compras a crédito.

<details>
<summary>Leer los seis conceptos en texto</summary>

## 7.0. Evita double dipping

```text
Unidades usadas por combo
≠
unidades disponibles automáticamente para otro beneficio
```

---

## 7.1. Mantiene trazabilidad comercial

Cada ajuste conoce:

- promoción;
- tipo;
- unidades;
- monto;
- origen.

---

## 7.2. Permite explicar el precio

El cliente puede recorrer:

```text
Precio base
→ beneficio comercial
→ obsequios
→ impuestos
→ total
```

---

## 7.3. Calcula correctamente impuestos

La base gravable se determina después de los descuentos comerciales.

---

## 7.4. Evalúa crédito sobre el valor correcto

No se consulta crédito contra subtotal bruto, sino contra el valor final que efectivamente se intentará confirmar.

---

## 7.5. Protege Quote = Order

El snapshot se crea únicamente una vez que el cálculo terminó.

```text
Pricing completo
→ Quote Snapshot
→ Confirm Order sin recalcular
```

---

</details>

# 8. Stack tecnológico

Monolito modular con puertos y adaptadores.

<img src="images/logos/go.svg" alt="Logo de Go" width="48" height="48"> **Go** — API REST, motor de precios y worker.

<img src="images/logos/mongodb.svg" alt="Logo de MongoDB" width="48" height="48"> **MongoDB** — Persistencia, transacciones, snapshots y outbox.

<img src="images/logos/googlecloud.svg" alt="Logo de Google Cloud Pub/Sub" width="48" height="48"> **Google Cloud Pub/Sub** — Emulador local para eventos OrderConfirmed.

<img src="images/logos/docker.svg" alt="Logo de Docker Compose" width="48" height="48"> **Docker Compose** — Servicios y pruebas reproducibles.

<img src="images/logos/flutter.svg" alt="Logo de Flutter" width="48" height="48"> **Flutter** — Frontend adaptable y golden tests visuales.

<img src="images/logos/dart.svg" alt="Logo de Dart" width="48" height="48"> **Dart** — Lenguaje del frontend Flutter.

<img src="images/logos/nginx.svg" alt="Logo de nginx" width="48" height="48"> **nginx** — Servidor web y proxy hacia la API.

<img src="images/logos/githubactions.svg" alt="Logo de GitHub Actions" width="48" height="48"> **GitHub Actions** — Tests, sintaxis, complejidad, seguridad y Pages.

<img src="images/logos/playwright.svg" alt="Logo de Playwright" width="48" height="48"> **Playwright** — Recorridos de compra en Chrome.

<img src="images/logos/postman.svg" alt="Logo de Postman" width="48" height="48"> **Postman** — Colección para probar la API.

# 9. Estructura de proyecto actual

```text
marketplace-multitenant-b2b-challenge/
├── .github/workflows/       # CI y publicación de documentación
├── backend/
│   ├── api/                 # HTTP API
│   ├── worker/              # Outbox, Pub/Sub, ERP y PUSH
│   ├── cmd/seed/            # Datos de demostración
│   ├── internal/
│   │   ├── application/    # Cotización y confirmación
│   │   ├── config/         # Configuración de ejecución
│   │   ├── domain/         # Entidades, Money y errores
│   │   ├── http/           # Router y validación del contrato
│   │   ├── integration/    # Adaptadores ERP y PUSH
│   │   ├── messaging/      # Outbox y consumidor Pub/Sub
│   │   ├── ports/          # Interfaces
│   │   ├── pricing/        # Motor de reglas y pruebas
│   │   ├── repository/mongo/
│   │   ├── testkit/        # Dobles para pruebas
│   ├── contracts/          # OpenAPI, eventos y Postman
│   ├── testdata/           # Seed y fixtures
│   ├── tests/integration/
├── frontend/
│   ├── lib/                # UI, modelos, API y componentes
│   ├── test/goldens/       # Referencias visuales
│   ├── e2e/                # Playwright
│   ├── tool/               # Herramientas de documentación
├── Documentation/          # Sitio, ADR e imágenes
├── ci/                     # Herramientas y reglas de calidad
├── docker-compose.yml
```

# 10. Flujo técnico resumido

![Flujo técnico resumido](images/arquitectura.svg)

La API devuelve 201 tras el commit; ERP/PUSH se procesan de forma asíncrona.

---

![Worker: suscripciones ERP/PUSH, reintentos y DLQ](images/worker_delivery.svg)

Bandeja local de correo: http://localhost:8025. Un tópico de negocio y dos suscripciones; DLQ durable en MongoDB.

# 11. Criterios de éxito

La solución se considera satisfactoria si demuestra:

```text
✓ Pricing correcto y explicable
✓ Golden Tests ejecutables
✓ Quote.total == Order.total
✓ Idempotencia
✓ Crédito consistente
✓ Sin floating point para Money
✓ Multi-país configurable
✓ 201 solo después de persistir
✓ OrderConfirmed publicado solo luego de confirmar
✓ Worker desacoplado para ERP y Push
✓ Código ejecutable con un comando simple
✓ README y ADR claros
```

---

# 12. Limitaciones

Para mantener foco en el problema de negocio:

- No se desplegará Kubernetes.
- No se implementará Kafka.
- No se diseñará infraestructura cloud productiva.
- No se implementará un motor genérico de promociones ilimitado.
- No se implementará un ERP real.
- No se implementará un proveedor Push real.

---

# 13. Promociones de prueba

Catálogo demostrativo basado en [backend/testdata/seed.json](../backend/testdata/seed.json) y en las reglas implementadas en [el motor de precios](../backend/internal/pricing/engine.go). Son datos sintéticos para evaluar la prueba técnica; no son ofertas comerciales reales.

## 13.0. Alcance y condiciones comunes

Hay **12 promociones por país**: 3 combos, 6 descuentos por escala y 3 reglas de regalo. Están disponibles para `tenant-demo` en Perú (PE/PEN), Chile (CL/CLP), Colombia (CO/COP), Ecuador (EC/USD), Guatemala (GT/GTQ) y Argentina (AR/ARS). Para reproducirlas en la tienda se utiliza el cliente demo `CUSTOMER-001`.

La vigencia configurada va desde `2020-01-01T00:00:00Z` (inclusive) hasta `2100-01-01T00:00:00Z` (exclusive). Las promociones originales `COMBO-001`, `SCALE-001` y `GIFT-001` tienen prioridad 1; las restantes, prioridad 2. Todas tienen la lista de incompatibilidades vacía. Dentro de cada tipo se ordenan por prioridad ascendente y después por ID.

El cálculo aplica **combo → escala → regalo → impuestos → total → evaluación de crédito**. Las unidades incluidas en un combo no reciben además descuento por escala. Los regalos sí cuentan todas las unidades del SKU que los activa, incluso las incluidas en combos. Las promociones aplican tanto al contado como a crédito; el crédito se valida sobre el total final.

## 13.1. Combos

| ID | Composición de un combo | Ahorro sobre el precio bruto de sus productos |
| --- | --- | --- |
| `COMBO-001` | 2 × `SKU-001` — Bebida original + 1 × `SKU-002` — Bebida selección | 25% |
| `COMBO-002` | 2 × `SKU-003` — Agua mineral + 1 × `SKU-005` — Jugo de naranja | 15% |
| `COMBO-003` | 2 × `SKU-007` — Té de durazno + 1 × `SKU-011` — Soda de limón | 15% |

Los siguientes importes son el **precio de un combo antes de impuestos**, en la moneda indicada. Se leen del seed; no representan conversiones de divisas.

| País / moneda | COMBO-001 | COMBO-002 | COMBO-003 |
| --- | --- | --- | --- |
| PE / PEN | 3,00 | 6,80 | 18,70 |
| CL / CLP | 3.000 | 6.800 | 18.700 |
| CO / COP | 3,00 | 6,80 | 18,70 |
| EC / USD | 3,00 | 6,80 | 18,70 |
| GT / GTQ | 3,00 | 6,80 | 18,70 |
| AR / ARS | 3,00 | 6,80 | 18,70 |

El combo se repite tantas veces como permita su componente más escaso. Ejemplo: 4 unidades de `SKU-003` y 2 de `SKU-005` forman dos `COMBO-002`. Si falta un componente, no se aplica el combo y las unidades se evalúan con las demás reglas.

## 13.2. Descuentos por escala

| ID | Productos elegibles | Tramos de unidades elegibles |
| --- | --- | --- |
| `SCALE-001` | `SKU-001` — Bebida original | 1–5: 0%; 6–10: 5%; Desde 11: 10% |
| `SCALE-002` | Familia `water`: `SKU-003` — Agua mineral; `SKU-004` — Agua con gas | 1–5: 0%; 6–11: 5%; 12–23: 10%; Desde 24: 15% |
| `SCALE-003` | Familia `juice`: `SKU-005` — Jugo de naranja; `SKU-006` — Jugo tropical | 1–5: 0%; 6–11: 5%; 12–23: 10%; Desde 24: 15% |
| `SCALE-004` | Familia `tea`: `SKU-007` — Té de durazno; `SKU-008` — Té de limón | 1–5: 0%; 6–11: 5%; 12–23: 10%; Desde 24: 15% |
| `SCALE-005` | Familia `energy`: `SKU-009` — Energética clásica; `SKU-010` — Energética sin azúcar | 1–5: 0%; 6–11: 5%; 12–23: 10%; Desde 24: 15% |
| `SCALE-006` | Familia `soda`: `SKU-011` — Soda de limón; `SKU-012` — Soda de pomelo | 1–5: 0%; 6–11: 5%; 12–23: 10%; Desde 24: 15% |

Los tramos son **acumulativos y marginales**: alcanzar el 15% no aplica ese porcentaje a todo el carrito. Se descuenta únicamente la cantidad que cae en cada tramo. En las escalas por familia se suman las unidades elegibles restantes después de los combos y se asignan los tramos por SKU ascendente, sin depender del orden del carrito.

Ejemplos en Perú, antes de impuestos: 12 unidades de `SKU-001` generan 5 unidades al 0%, 5 al 5% y 2 al 10%: descuento total **0,45 PEN**. En la familia de aguas, 12 unidades de `SKU-003` y 12 de `SKU-004` generan 5 al 0%, 6 al 5%, 12 al 10% y 1 al 15%: descuento total **4,55 PEN**. El segundo carrito no forma combo porque no contiene `SKU-005`.

## 13.3. Regalos

| ID de la promoción | Condición | Regalo por cada múltiplo completo |
| --- | --- | --- |
| `GIFT-001` | 6 unidades de `SKU-001` — Bebida original | 1 × `GIFT-001` — Obsequio de la casa |
| `GIFT-002` | 6 unidades de `SKU-005` — Jugo de naranja | 1 × `GIFT-002` — Vaso reutilizable |
| `GIFT-003` | 12 unidades de `SKU-009` — Energética clásica | 1 × `GIFT-003` — Bolsa térmica |

Cantidad de regalos = parte entera de (unidades del SKU elegible / mínimo requerido) × cantidad de regalo configurada. Así, 12 unidades de `SKU-001` dan 2 obsequios de la casa; 12 de `SKU-005`, 2 vasos; y 24 de `SKU-009`, 2 bolsas térmicas. No es necesario agregar el regalo al carrito: aparece al cotizar.

El valor de referencia del regalo es informativo, no se suma al total a pagar. El seed configura `taxGifts: false` en los seis países; por eso el impuesto del regalo es cero en estos escenarios. Los productos comprados sí se gravan según la configuración del país.

## 13.4. Carritos para reproducir las promociones

En el catálogo, usa las selecciones de «¿Por dónde empezar?» o ajusta manualmente las cantidades. Elige el país y el pago al contado para aislar las promociones del cupo de crédito, pulsa **Cotizar pedido** y revisa «Cada beneficio, a la vista» junto al resumen. Una nueva selección reemplaza el carrito anterior.

| Selección de la tienda / prueba manual | Carrito | Beneficio esperado |
| --- | --- | --- |
| Por volumen · 12 uds. | 12 × `SKU-001` | `SCALE-001` y 2 × `GIFT-001`. |
| Combo + volumen | 8 × `SKU-001` + 1 × `SKU-002` | 1 × `COMBO-001`; escala sobre las 6 unidades restantes de SKU-001 (solo una al 5%); 1 × `GIFT-001` por las 8 unidades compradas. |
| Con obsequio · 6 uds. | 6 × `SKU-001` | 1 × `GIFT-001` y escala al 5% sobre la sexta unidad. |
| Aguas · escala por familia | 12 × `SKU-003` + 12 × `SKU-004` | `SCALE-002` sobre 24 unidades de la familia water; la última entra al tramo del 15%. |
| Jugos + vaso de regalo | 6 × `SKU-005` | 1 × `GIFT-002` y `SCALE-003` al 5% sobre la sexta unidad. |
| Energéticas + bolsa térmica | 12 × `SKU-009` | 1 × `GIFT-003` y `SCALE-005`: 6 unidades al 5% y 1 al 10%. |
| Combo refrescante | 2 × `SKU-003` + 1 × `SKU-005` | 1 × `COMBO-002`; esas unidades no reciben escala. |
| Manual: segundo combo nuevo | 2 × `SKU-007` + 1 × `SKU-011` | 1 × `COMBO-003`. |
| Manual: escala de té | 24 × `SKU-007` | `SCALE-004`; una unidad llega al tramo del 15%. |
| Manual: escala de soda | 24 × `SKU-011` | `SCALE-006`; una unidad llega al tramo del 15%. |

Para comprobar límites, prueba 5 y 6 unidades de `SKU-001` o `SKU-005`, y 11 y 12 de `SKU-009`: el regalo aparece al alcanzar su mínimo. En las escalas por familia, compara 11/12 y 23/24 unidades para observar el cambio de tramo. Si también agregas productos de un combo, las cantidades disponibles para escala pueden disminuir.

La cobertura de las nueve promociones nuevas en los seis países está en [seed_test.go](../backend/internal/pricing/seed_test.go). Los recorridos de navegador y el detalle de beneficios se documentan en [Validación del frontend](../frontend/VALIDATION.md).
