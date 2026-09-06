# Validación local

Ejecutada el 5 de septiembre de 2026 (hora de Chile), sobre el código de `168aceb`, con Docker Desktop 4.88.1, Engine 29.7.2, Compose v5.4.0 y contenedores Linux amd64. Go dentro de la imagen: 1.25.14. No se requirió instalar Go en Windows.

## Resultados

| Comprobación | Resultado |
|---|---|
| `docker compose up --build -d` | PASS: imágenes compiladas, volumen nuevo, Mongo primario, seed con exit 0 y API iniciada |
| `go test -count=1 ./...` dentro del contenedor tests | PASS: todos los paquetes con pruebas |
| `docker compose --profile test run --build --rm tests` | PASS: suite completa con `-race -tags=integration -count=1`, sin carreras reportadas |
| `go vet ./...` dentro del contenedor tests | PASS, sin diagnósticos |
| `GET /health` y `GET /ready` | PASS: `ok` y `ready` después de reiniciar la API tras la recreación del emulador |
| `scripts/smoke.ps1` contra la API Docker | PASS: quote, confirmación, replay y consulta inmediata; total conservado de 1363 unidades menores (13.63 PEN) |
| Consulta directa a Mongo del pedido smoke | PASS: evento publicado, ERP y Push completados, exactamente dos recibos persistidos |

Las pruebas de integración verificaron 100 confirmaciones simultáneas con un solo pedido y débito, competencia por crédito y rollback, snapshot y aislamiento por país, conflicto de payload con la misma clave, publicación/consumo real por el emulador y deduplicación incluso al perder el marcador local del worker. Los fallos de destinos y recuperación de outbox también están cubiertos por dobles en la suite; no se contactaron servicios ERP/Push reales.

Pedido smoke: `e4f125b9-c525-4aef-98c5-b93f016be50a`.
Evento: `96019503-4ba5-4680-bc4c-6fa3987831a1`.

## Benchmark

`go test ./internal/pricing -run '^$' -bench . -benchmem`

CPU reportada: AMD Ryzen 7 5800U with Radeon Graphics.

```text
BenchmarkPricing100Lines20Promotions-16  4610  259431 ns/op  170193 B/op  1077 allocs/op
```

Aproximadamente 0.259 ms por cálculo de 100 líneas y 20 promociones. Es una medición local del motor, no de latencia HTTP ni un SLA de producción.

## Repetir la validación

Desde `backend`, levantar el entorno y, una vez disponible, ejecutar:

```sh
docker compose up --build -d
docker compose --profile test run --build --no-deps --rm tests
docker compose --profile test run --no-deps --rm tests go test -count=1 ./...
docker compose --profile test run --no-deps --rm tests go vet ./...
docker compose --profile test run --no-deps --rm tests go test ./internal/pricing -run '^$' -bench . -benchmem
```

En PowerShell:

```powershell
Invoke-RestMethod http://localhost:8080/ready
powershell -NoProfile -ExecutionPolicy Bypass -File scripts/smoke.ps1
```

El smoke crea un pedido y consume crédito del cliente demo. Las pruebas de integración usan bases aleatorias independientes que eliminan al finalizar.

## Incidencia observada y límite del emulador

Al ejecutar el perfil test sin `--no-deps`, Compose recreó Mongo y Pub/Sub. Mongo conservó sus datos en el volumen; el emulador perdió la suscripción en memoria, y `/ready` devolvió 503. `docker compose restart api` recreó los recursos y restableció `/ready` y el procesamiento. Para probar sobre un entorno ya activo, usar `--no-deps`.

Recrear o reiniciar el emulador puede perder mensajes ya publicados que todavía no se consumieron. La outbox recupera eventos pendientes de publicación, pero no reproduce automáticamente eventos marcados como publicados. Esta validación no demuestra durabilidad del emulador ante pérdida de proceso ni comportamiento de servicios cloud productivos.

El entorno Docker quedó levantado con la API disponible en `http://localhost:8080`.
