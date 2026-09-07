# API

Aplicación HTTP independiente. Desde `backend`: `go run ./api` o `make run`.

Cotiza, consulta snapshots y confirma/consulta pedidos. La confirmación persiste pedido, crédito, idempotencia y outbox en una transacción. No publica mensajes ni llama a ERP/PUSH.

Configuración: `HTTP_PORT`, `MONGO_URI`, `MONGO_DATABASE`, `QUOTE_TTL`, `LOG_LEVEL`. `/health` indica que el proceso está activo; `/ready` comprueba MongoDB. Puede confirmar sin broker ni worker; los eventos quedan pendientes en outbox.

Docker: servicio `api`, target `runtime` del Dockerfile compartido. El dominio y repositorios permanecen en `backend/internal`, dentro de un solo módulo Go.
