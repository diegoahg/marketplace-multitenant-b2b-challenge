# Revisión final contra la prueba técnica

Fecha: 7 de septiembre de 2026. Fuente: `Prueba_Tecnica_Tech_Lead_1.pdf`, tres páginas. Se contrastaron sus requisitos con el código y las pruebas del workspace. El documento se usó como criterio de evaluación; los cambios posteriores del usuario definen las extensiones de worker, DLQ, correo y diseño.

**Conclusión:** los requisitos técnicos A1, A2 y A3 y el frontend opcional tienen implementación y evidencia local. Postman sigue siendo compatible. La entrega remota no se considera cerrada mientras los últimos cambios no estén publicados y verificados por GitHub Actions. No se asigna un porcentaje oficial: el PDF no proporciona pesos de evaluación.

## Trazabilidad del PDF

| Requisito | Implementación y evidencia | Resultado |
|---|---|---|
| A1: desglose de líneas, descuentos con origen, regalos, impuestos y total | `internal/pricing`, `POST /quotes`, contratos JSON y fixtures GT01–GT05 | Cumple |
| A1: coexistencia y orden justificado | COMBO → SCALE → GIFT → impuestos → crédito; escala por familia con asignación determinista; README y ADR | Cumple |
| A1: elegibilidad de crédito | Quote informativa y validación atómica al confirmar; tests de límites y GT05/GT08 | Cumple |
| A1: cotizado = confirmado pese a cambios | Snapshot persistido; confirmación no recalcula; GT06 y prueba Mongo de snapshot | Cumple |
| A1: dinero exacto y configuración por país | Enteros, intermedios `math/big`, escala y rounding configurables, tasas desde datos; GT09/GT10 | Cumple |
| Seis golden obligatorios | `backend/testdata/golden/GT01.json`–`GT06.json`, motor y aplicación | Cumple |
| A2: idempotencia y no duplicar crédito | Índice por tenant/clave, hash del request, transacción y prueba de 100 confirmaciones concurrentes | Cumple |
| A2: 201 implica orden persistida y consultable | Commit antes de responder; `GET /orders/{id}` y Postman | Cumple |
| A2: efectos recuperables y consistencia documentada | Outbox, dos suscripciones, deduplicación por destino, reintentos, DLQ durable y alertas; efectos dummy | Cumple con contrato idempotente exigido a receptores reales |
| A3: ADR máximo una página | `adr.html` y `ADR.pdf`: decisiones, alternativas, exclusiones, code review, riesgo y mitigación | Cumple; una página A4 verificada |
| Front opcional: desglose y tres defectos de UX | Flutter; textos largos a ancho mediano, insets/teclado, imágenes 4:3 contain; 43 tests y ocho golden visuales | Cumple en Flutter Web y simulación de insets |
| Entrega: ejecución, README y Git | Compose y comandos documentados; código local comprobado | Publicación de últimos cambios y CI remoto pendientes |

## Comprobaciones y alcance

- Postman/Newman: **28 requests y 71 aserciones aprobadas**. Cotiza los seis países, confirma, consulta, verifica replay, crédito insuficiente y errores de scope/contrato.
- Flutter: **43 tests aprobados**, ocho PNG comparados y analizador sin incidencias.
- Go: pruebas unitarias, seis golden de negocio e integración con `-race -tags=integration -count=1`. Las pruebas de Mongo ejercen atomicidad, concurrencia y recuperación.
- Worker: `worker-smoke.mjs` verifica API disponible sin broker/worker y recuperación del outbox. `dlq-smoke.mjs` verifica los dos destinos, recuperación transitoria, seis fallos, DLQ y correo después de una interrupción SMTP. Evidencia detallada en `backend/VALIDATION.md`.
- Postman no sustituye las pruebas de concurrencia ni inspecciona DLQ/ERP/PUSH mediante una API inexistente. El correo de prueba se consulta en Mailpit, `http://localhost:8025`.

En esta revisión se corrigió un aislamiento adicional: un payload inválido que copiaba el ID de un evento válido ya no puede consumir su presupuesto de retries ni contaminar su estado DLQ. Se añadió una prueba de regresión.

## Límites explícitos

- ERP/PUSH son dummies; en producción los receptores deben deduplicar por `eventId:destino`. No se promete exactly-once distribuido.
- La DLQ es `dead_letters` en MongoDB, no un tercer tópico de negocio. Conserva evidencia y requiere revisión/redrive supervisado; no se reejecuta automáticamente un efecto terminal.
- El SMTP es local y el correo puede duplicarse si el proceso cae entre aceptación SMTP y persistencia de éxito. Mailpit no envía estos avisos a terceros.
- Los impuestos de CO/EC/GT/AR son datos sintéticos en cero; no hay conversión cambiaria. La configuración del motor permanece parametrizada.
- El replay conserva todos los campos de negocio; fechas se comparan a precisión de milisegundos de BSON. No se afirma igualdad byte a byte de la respuesta inicial.
- No se han validado builds nativos ni teclados físicos Android/iOS. Los límites de dedicación temporal indicados en el PDF no se pueden certificar a partir del código.

## Arquitectura actual

![Arquitectura actual de MariposaMarket](images/arquitectura.svg)

[Resumen](Readme.md) · [Dominio](dominio.md) · [Casos](casos.md) · [ADR de una página](adr.html)
