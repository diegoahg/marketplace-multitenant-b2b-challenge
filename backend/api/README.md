# API

Aplicación HTTP independiente. Desde `backend`: `go run ./api` o `make run`.

Cotiza, consulta snapshots y confirma/consulta pedidos. La confirmación persiste pedido, crédito, idempotencia y outbox en una transacción. No publica mensajes ni llama a ERP/PUSH.

Configuración: `HTTP_PORT`, `MONGO_URI`, `MONGO_DATABASE`, `QUOTE_TTL`, `LOG_LEVEL`. `/health` indica que el proceso está activo; `/ready` comprueba MongoDB. Puede confirmar sin broker ni worker; los eventos quedan pendientes en outbox.

Docker: servicio `api`, target `runtime` del Dockerfile compartido. El dominio y repositorios permanecen en `backend/internal`, dentro de un solo módulo Go.


`GET /orders/{id}/details` conserva la consulta original de la orden y agrega su snapshot de cotización y los estados independientes de ERP/PUSH. Requiere los mismos headers de tenant, país y cliente; valida la propiedad antes de leer las entregas. `deliveries: null` significa información no disponible, nunca una entrega confirmada. Contrato: `backend/contracts/openapi.json`.

Logs JSON informativos: `quote persisted`, `order confirmation available` y `order transaction committed` (después del commit). Incluyen identificadores de cotización/pedido y, en HTTP, `traceId`; una confirmación puede ser un replay idempotente. Los errores de transacción se registran con `order transaction failed`.
