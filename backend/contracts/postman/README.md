# Postman — MariposaMarket

Importa estos dos archivos en Postman:

- `MariposaMarket.postman_collection.json`: colección v2.1 con 28 requests y aserciones automáticas.
- `Docker-local.postman_environment.json`: entorno local, sin secretos.

Desde la raíz del repositorio, levanta los servicios con `docker compose up --build -d`. Selecciona el entorno **MariposaMarket - Docker local** y ejecuta la colección completa con **Run collection**, en el orden incluido. También puedes enviar las requests manualmente respetando la numeración. Una confirmación inicial debe hacerse antes de que venza la cotización (15 minutos por defecto).

## Variables

| Variable | Valor inicial | Uso |
|---|---|---|
| `baseUrl` | `http://localhost:8080` | API directa, sin barra final. También admite `http://localhost:3000/api` mediante nginx. |
| `tenantId` | `tenant-demo` | Tenant del seed. |
| `country` | `PE` | País del flujo principal: PE, CL, CO, EC, GT o AR. |
| `customerId` | `CUSTOMER-001` | Cliente del seed. |

Las requests guardan `quoteId`, `orderId`, `idempotencyKey` y snapshots en variables de colección. No crees variables de entorno con esos nombres: ocultarían los valores generados. La primera request de compra renueva la clave y limpia los resultados anteriores; el replay conserva la misma clave y payload.

## Cobertura

- Health del proceso API y readiness de MongoDB. La API permanece disponible sin worker ni Pub/Sub.
- Cotización con escalas acumuladas, origen de descuentos, obsequios, impuestos y aritmética exacta en unidades menores.
- Lectura del snapshot, compra al contado, consulta del pedido y conservación del total.
- Replay equivalente en campos de negocio y fechas a milisegundos, conflicto al cambiar el payload y rechazo de otra clave para una cotización ya confirmada.
- Cotizaciones en PEN, CLP, COP, USD, GTQ y ARS.
- Cotización a crédito no elegible y rechazo de la confirmación sin saldo.
- Errores de contexto, cantidad, SKU duplicado/inexistente, campo desconocido, clave ausente y aislamiento por tenant, cliente y país.

Requiere los precios y promociones originales del seed. Los cuatro países nuevos tienen impuestos sintéticos en cero y precios nominales; no se convierten divisas. El caso de crédito insuficiente usa PE y presupone su saldo de demostración. Volver a ejecutar el seed no restaura datos modificados.

Cada ejecución crea cotizaciones y un pedido **CASH**; no descuenta crédito. La colección no elimina datos. ERP/PUSH asíncronos, carreras entre confirmaciones y recuperación del broker se prueban en `backend/tests/integration`, ya que no existe una API pública para consultar esos efectos.

Revalidación del 7 de septiembre de 2026: **28 requests y 71 aserciones aprobadas** después de separar API/worker y agregar dos suscripciones, DLQ y correo. Los contratos HTTP no cambiaron. Para los efectos asíncronos, ejecutar desde la raíz `node backend/scripts/worker-smoke.mjs` y `node backend/scripts/dlq-smoke.mjs`; la bandeja de prueba está en `http://localhost:8025`.

Las comparaciones de snapshots y replay normalizan únicamente `createdAt` y `expiresAt` a milisegundos: Go entrega nanosegundos en la respuesta inicial y BSON conserva milisegundos. Los demás campos se comparan íntegramente, sin tolerancias monetarias. Por esta diferencia de fechas el replay actual es equivalente a precisión de persistencia, pero no idéntico byte a byte.

## Ejecutar sin la interfaz de Postman

Desde la raíz, con Node.js y el backend disponible:

```sh
npx --yes --package=newman@6.2.1 newman run backend/contracts/postman/MariposaMarket.postman_collection.json -e backend/contracts/postman/Docker-local.postman_environment.json --timeout-request 15000
```

Newman devuelve un código distinto de cero si falla una aserción. Para probar el flujo principal en otro país, agrega `--env-var country=GT`. Las seis requests de la carpeta de países siempre prueban sus países explícitos.
