# ADR 001 — Precio confirmado y efectos externos recuperables

Estado: aceptado para la prueba técnica.

**Contexto.** Un pedido combina beneficios comerciales y consume crédito compartido. Los reintentos HTTP y las entregas duplicadas son normales. El precio visto por el cliente debe respetarse aunque las reglas cambien.

**Decisiones.** Go y monolito modular con puertos/adaptadores mantienen reglas comprobables sin HTTP o infraestructura. MongoDB almacena snapshots y documentos de reglas, y usa un replica set para transacciones. Money emplea unidades menores int64, intermedios de precisión arbitraria, validación de moneda/escala y rounding configurable; JSON expresa el entero como string. Quote Snapshot persiste el breakdown completo y la confirmación nunca recalcula precios.

Idempotencia durable con índice `(tenantId,key)` y hash del request devuelve la respuesta original o 409 si cambia. Una quote no puede confirmar dos pedidos, incluso con keys distintas. La misma transacción reserva quote, consume crédito con condición atómica `available >= total`, crea Order, idempotencia y outbox. Write concern majority y lectura snapshot evitan estados parciales.

Pub/Sub local ofrece entrega at-least-once. El publisher reintenta outbox y solo marca publicado después de recibir confirmación. El worker asíncrono gestiona ack/nack y progreso por ERP/Push, con leases recuperables. Los mocks deduplican mediante recibos durables; los receptores HTTP deben implementar `Idempotency-Key` para cubrir la caída entre efecto remoto y progreso local.

**Alternativas descartadas.** Microservicios y Kafka agregan operación sin necesidad demostrada. Recalcular quote rompe la promesa de precio. Llamar ERP dentro del request acopla disponibilidad y latencia. `float64` pierde precisión monetaria. Publicar tras guardar Order sin outbox puede perder eventos.

**Trade-offs.** Transacciones requieren replica set y tienen costo de contención. Un nodo local no da alta disponibilidad. No existe exactly-once distribuido. Un worker secuencial facilita razonamiento pero limita throughput. La política de combos excluye escala sobre las mismas unidades. No se implementa todavía una DLQ.

**Riesgo principal y mitigación.** Efectos externos duplicados ante timeout o caída: clave durable por eventId/destino, checkpoints por destino y pruebas de redelivery/caída. En producción hay que verificar ese contrato con los receptores, agregar identidad real, métricas, backoff/DLQ y réplicas.

**Code review.** Revisar alcance tenant/país/cliente, índices únicos, moneda/overflow/redondeo, asignación de unidades promocionales, frontera transaccional y rollback, replay después de expiración, ack solo tras ambos destinos, recuperación de leases, cancelación de goroutines y que los tests de concurrencia ejerzan Mongo real.
