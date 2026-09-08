# Reconstrucción de la demo

Leer este documento cuando se construya nuevamente el proyecto o se revise su alcance completo. Resume el encargo original y las extensiones posteriores; los ejemplos de contratos históricos no deben sustituir los contratos vigentes del código.

## Componentes

- Backend Go modular con dominio y motor puros, casos de uso, puertos, adaptadores Mongo/HTTP/PubSub e integraciones reemplazables. Inyección manual y context para cancelación/cierre.
- API en backend/api, worker en backend/worker y carga de datos en backend/cmd/seed. Mantener el módulo compartido y procesos independientes.
- Mongo replica set local; índices de scope e idempotencia; catálogos, clientes, promociones, cotizaciones, pedidos, crédito, outbox, estado procesado y DLQ durables.
- Pub/Sub emulator: tópico orders-confirmed, suscripciones orders-confirmed-erp y orders-confirmed-push. Dummies ERP/PUSH y Mailpit para correo local.
- Flutter Web: catálogo → carrito → cotización → resumen → confirmación → historial y detalle. Importes calculados por backend, sin un segundo motor de precios en el cliente.
- Docker Compose para api, worker, seed, mongo, pubsub-emulator, mailpit y frontend; perfiles separados de pruebas. No incorporar Kubernetes, Kafka, Redis, service mesh o cloud productivo salvo un nuevo encargo.

## Dominio y contratos

Modelar Money, Product, Customer, Cart, Promotion, Quote, Order, CreditAccount y OrderConfirmed. Tenant, país y cliente delimitan todas las operaciones y consultas, incluido el seguimiento de entregas. Headers y payload deben respetar el scope del contrato vigente; no son autenticación productiva.

Money expone amount como string de unidades menores, currency y scale. La cotización explica líneas, ajustes con origen, subtotal, descuentos, regalos con SKU/cantidad, base gravable, impuestos, total y elegibilidad de crédito. Regalos a precio cero mantienen su tratamiento tributario configurable. Redondear de forma explícita y determinista; evitar overflow en intermedios.

Las escalas son acumulativas: por ejemplo 1–5 al 0 %, 6–10 al 5 % y 11+ al 10 % afecta cada tramo, no todo el carrito. Los combos reservan sus unidades. Compatibilidad, prioridad, vigencia y desempate deben ser deterministas; para escalas por familia, el orden de entrada del carrito no cambia la asignación.

Contratos mínimos: POST /quotes, GET /quotes/{id}, POST /orders con Idempotency-Key, GET /orders/{id}; conservar catálogo y GET /orders/{id}/details para el front. El detalle incluye orden, cotización y entregas ERP/PUSH; ausencia de seguimiento no implica confirmación externa.

Errores HTTP consistentes para petición inválida, producto/quote/orden inexistente, quote inválida o expirada, crédito insuficiente, conflicto de idempotencia y fallo interno. No exponer stack traces. Mantener ejemplos JSON, esquema del evento, colección Postman y documentación en backend/contracts; OpenAPI solo si se crea y mantiene consistente.

## Atomicidad y recuperación

- Snapshot inmutable y TTL explícito. Confirmar desde el snapshot, comprobar scope y crédito disponible actual.
- Índice único por tenant y clave de idempotencia, hash del request y respuesta persistida. Cien solicitudes concurrentes con la misma clave producen una sola orden y un débito.
- Condición atómica de saldo: dos órdenes compitiendo por un saldo que no cubre ambas no pueden dejarlo negativo. Las compras al contado no consumen crédito.
- Transacción Mongo: orden, débito cuando corresponda, idempotencia y outbox; commit antes del 201.
- Outbox recuperable ante broker caído y caída entre publicar y marcar publicado. Reconciliar publicaciones incompletas sin repetir efectos terminados.
- Entrega at-least-once; deduplicación por eventId/destino. Estado durable por destino. Seis fallos llevan a DLQ en Mongo; mensajes inválidos no contaminan el presupuesto de un evento válido.
- Reintentar avisos SMTP pendientes al recuperarse Mailpit. El correo puede duplicarse si el proceso cae después de enviarlo y antes de persistir éxito. Redrive de DLQ supervisado, no automático.

## Datos y experiencia de demostración

Mantener seis países PE/PEN, CL/CLP, CO/COP, EC/USD, GT/GTQ y AR/ARS, cada uno con configuración propia. La demo ampliada dispone de 12 productos comprables y tres regalos por país, junto con combos, escalas y obsequios. Tomar cantidades, umbrales e importes concretos de backend/testdata/seed.json; no inventarlos. Las tasas nuevas de CO/EC/GT/AR son sintéticas en cero; los precios no representan conversión cambiaria.

Mostrar promociones cerca del resumen, detalle de orden al pulsar y seguimiento ERP/PUSH real dentro de la demo. El aviso sobre el carácter demostrativo del historial es obligatorio. Probar textos largos, viewport mediano/móvil, teclado/insets y proporción de imágenes. Separar ejemplos de promociones en la documentación y explicar cómo activarlos.

## Evidencia de aceptación

| Caso | Resultado que debe comprobarse |
|---|---|
| GT01 | Escala con dos tramos y desglose correcto |
| GT02 | Combo sin segundo descuento sobre las mismas unidades |
| GT03 | Regalo con SKU, cantidad y promoción |
| GT04 | Combo e impuestos calculados sobre la base correcta |
| GT05 | Crédito excedido: no elegible y confirmación rechazada |
| GT06 | Cambian precios/promociones después de cotizar y se conserva el total al confirmar |
| Adicionales | Concurrencia/idempotencia, saldo competido, redondeo, multi-país, evento solo tras commit y redelivery sin duplicar efectos |
| Integraciones | API disponible sin broker, outbox recuperado, destinos independientes, DLQ y recuperación SMTP |
| Frontend | Golden visuales, controles accesibles y E2E contra API real, incluido historial/detalle |

Cubrir carrito vacío, cantidades no positivas, SKU inexistente, promoción expirada, combo incompleto, límites de escala, excedentes de combo, múltiples candidatos, crédito exacto y menor por una unidad, quote expirada y reutilización conflictiva de clave. Comparar cifras y estados, no solo presencia de una promoción.

Mantener un benchmark orientativo de 100 líneas y 20 promociones, sin inventar SLA. Documentar arquitectura, orden de palancas con motivos de Producto/Marketing, consistencia, rounding, contratos, ejecución, límites y alternativas. ADR de una página y trazabilidad del PDF cuando esté disponible. La entrega debe poder reproducirse con README y un comando de suite; consultar las cifras de tests efectivamente ejecutados.
