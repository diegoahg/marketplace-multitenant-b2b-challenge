# Worker

Aplicación independiente de mensajería. Desde `backend`: `go run ./worker` o `make worker`. Docker: servicio y target `worker`, con reinicio `unless-stopped`.

Ejecuta cuatro bucles concurrentes:

1. Publica eventos pendientes del outbox. Reintenta fallos cada segundo y reconcilia publicaciones de más de un minuto con destinos incompletos, conservando `eventId`.
2. Consume `orders-confirmed-erp` y ejecuta el dummy ERP.
3. Consume `orders-confirmed-push` y ejecuta el dummy PUSH, sin esperar al ERP.
4. Consume la DLQ durable `dead_letters` de MongoDB y envía alertas por SMTP a Mailpit.

Un único tópico de negocio, `orders-confirmed`, distribuye cada evento a las dos suscripciones. Cada destino hace su propio ACK al completar su acción. Un fallo no hace ACK: se modifica el plazo de entrega y se guarda `nextAttempt` en MongoDB. Hay un intento inicial y **cinco reintentos**, con esperas mínimas de **1, 2, 4, 8 y 16 segundos**. La entrega efectiva puede tardar más por planificación del broker. Contador, última causa y próxima fecha sobreviven a reinicios; una republicación no reinicia el presupuesto.

Tras el sexto fallo, una transacción guarda el evento/error/destino/intentos en `dead_letters` y marca ese destino terminal. Solo después hace ACK del mensaje original. No se marca el efecto fallido como exitoso. La reconciliación del outbox excluye destinos completados o en DLQ; si el otro sigue pendiente, conserva su recuperación. Mensajes malformados también tienen una clave estable y terminan en DLQ.

Variables: `MONGO_URI`, `MONGO_DATABASE`, `PUBSUB_EMULATOR_HOST`, `PUBSUB_PROJECT_ID`, `PUBSUB_TOPIC`, `PUBSUB_ERP_SUBSCRIPTION`, `PUBSUB_PUSH_SUBSCRIPTION`, `ERP_BASE_URL`, `PUSH_BASE_URL`, `SMTP_ADDRESS`, `DLQ_MAIL_FROM`, `DLQ_MAIL_TO`, `LOG_LEVEL`. URLs de destinos vacías activan dummies con recibos durables. Para provocar fallos de prueba se inserta en `dummy_failures` un documento `{_id: "EVENT_ID:erp", remaining: 6}` (o `:push`); cada fallo consume una unidad, solo en modo dummy.

La bandeja local está en **http://localhost:8025**. El correo contiene pedido, evento, destino, causa, intentos y fecha. Si SMTP falla, el registro continúa pendiente y se reintenta después de 30 segundos. Un lease evita envíos simultáneos; `Message-ID` permanece estable. SMTP no garantiza deduplicación: una caída tras aceptar el correo y antes de guardar `emailedAt` puede repetir la alerta. Los registros se conservan para revisión; no hay redrive automático de DLQ.

No expone HTTP. Se observa con `docker compose logs -f worker`, `processed_events` y `dead_letters`. SIGTERM cancela los bucles, espera su finalización y cierra MongoDB. El adaptador SMTP es para la bandeja local sin TLS/autenticación; SMTP productivo y métricas de salud quedan fuera de esta prueba.

Desde la raíz, `node backend/scripts/worker-smoke.mjs` verifica la separación con Docker y destinos mock. Detiene temporalmente worker/broker y los vuelve a iniciar incluso si falla una aserción.

`node backend/scripts/dlq-smoke.mjs` crea tres pedidos CASH: ERP falla seis veces, PUSH falla seis veces y un tercer ERP se recupera tras dos fallos. Verifica independencia, DLQ y entrega de correos después de una caída de SMTP. Restaura worker/Mailpit al finalizar y conserva la evidencia.

La antigua suscripción `orders-confirmed-worker` ya no se consume. En un entorno anterior puede seguir existiendo hasta reiniciar el emulador; el outbox en Mongo conserva los pendientes. No se borran suscripciones ajenas automáticamente.
