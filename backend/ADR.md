# ADR 001 — Precio confirmado y efectos externos recuperables

Estado: aceptado para la prueba técnica.

**Separación de procesos.** `backend/api` atiende HTTP y persiste la transacción con outbox; `backend/worker` publica/reconcilia pendientes y consume confirmaciones para ERP/PUSH. Comparten dominio y repositorios, pero arrancan y se reinician por separado. Se descarta mantener el worker embebido en la API para que una caída de mensajería no impida confirmar pedidos. El coste es operar y observar dos procesos; readiness de la API solo comprueba MongoDB.

Síntesis actualizada con contexto de negocio, uso de IA y revisión: [ADR de una página](../Documentation/adr.html).

**Contexto.** Un pedido combina beneficios comerciales y consume crédito compartido. Los reintentos HTTP y las entregas duplicadas son normales. El precio visto por el cliente debe respetarse aunque las reglas cambien.

**Decisiones.** Go y monolito modular con puertos/adaptadores mantienen reglas comprobables sin HTTP o infraestructura. MongoDB almacena snapshots y documentos de reglas, y usa un replica set para transacciones. Money emplea unidades menores int64, intermedios de precisión arbitraria, validación de moneda/escala y rounding configurable; JSON expresa el entero como string. Quote Snapshot persiste el breakdown completo y la confirmación nunca recalcula precios.

Idempotencia durable con índice `(tenantId,key)` y hash del request devuelve la respuesta original o 409 si cambia. Una quote no puede confirmar dos pedidos, incluso con keys distintas. La misma transacción reserva quote, consume crédito con condición atómica `available >= total`, crea Order, idempotencia y outbox. Write concern majority y lectura snapshot evitan estados parciales.

Pub/Sub local ofrece entrega at-least-once: un tópico `orders-confirmed` y dos suscripciones independientes, `orders-confirmed-erp` y `orders-confirmed-push`. El worker ejecuta dummies por destino, con leases y ACK propio. El publisher conserva el outbox hasta la confirmación del broker. Los receptores HTTP deben respetar `Idempotency-Key` para cubrir la caída entre efecto remoto y progreso local.

**Reintentos y DLQ.** Un intento inicial más cinco reintentos, separados por 1/2/4/8/16 segundos, con contador y próxima ejecución en Mongo. No se hace ACK al fallar; se aplaza la entrega. Tras el sexto fallo, una transacción marca el destino terminal e inserta una entrada única en la cola Mongo `dead_letters`; después se confirma el mensaje fuente. Un cuarto bucle envía la alerta a Mailpit y reintenta SMTP fallido cada 30 segundos. No se borra la evidencia. Se descarta una suscripción compartida porque una entrega solo iría a un consumidor, y se descarta depender del conteo best-effort del broker para un presupuesto preciso y durable. La DLQ es una cola persistida en Mongo, no otro tópico Pub/Sub.

La reconciliación vuelve a publicar eventos de más de un minuto con algún destino pendiente, usando el mismo ID y contador. Los destinos completados o en DLQ son terminales. Tópico y ambas suscripciones se recuperan ante 404. Para escalas por familia se acumula el saldo post-combo y se asignan tramos por SKU ascendente, independiente del orden del carrito.

**Alternativas descartadas.** Microservicios y Kafka agregan operación sin necesidad demostrada. Recalcular quote rompe la promesa de precio. Llamar ERP dentro del request acopla disponibilidad y latencia. `float64` pierde precisión monetaria. Publicar tras guardar Order sin outbox puede perder eventos.

**Trade-offs.** Transacciones requieren replica set y tienen costo de contención. Cada destino consume secuencialmente, pero ambos avanzan en paralelo. La DLQ gestionada por la aplicación agrega estado y pruebas, a cambio de recuperación durable y un límite explícito. No hay redrive automático ni alta disponibilidad local. SMTP puede repetir una alerta si el proceso cae tras enviarla y antes de persistir éxito; se conserva Message-ID. La política de combos excluye escala sobre las mismas unidades.

**Riesgo principal y mitigación.** Efectos externos duplicados ante timeout o caída: clave durable por eventId/destino, checkpoints y pruebas de redelivery/caída. En producción hay que verificar ese contrato con los receptores, agregar identidad, métricas, redrive supervisado, SMTP con TLS/autenticación y réplicas.

**Code review.** Revisar alcance tenant/país/cliente, índices únicos, dinero y promociones, frontera transaccional, replay, ACK por destino solo después del efecto o de persistir DLQ, fechas y contadores de retry, recuperación de leases, aislamiento de los dummies y pruebas con Mongo/PubSub reales.
