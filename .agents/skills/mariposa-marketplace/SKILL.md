---
name: mariposa-marketplace
description: Preparar, reconstruir o retomar la demo MariposaMarket del desafío Marketplace B2B con Go y Flutter, preservando precios, pedidos, crédito e integraciones. Usar cuando se pida repetir esta app o dejar su entorno listo para otra prueba.
---

# MariposaMarket reutilizable

Sustituye el antiguo backend/instruccion.txt. El resultado es una app ejecutable, comprobable y documentada. Conserva las ampliaciones acordadas: Flutter Web, API y worker separados, historial de pedidos, ERP/PUSH de demostración, DLQ y correo local.

## Determinar el trabajo

- Localiza la raíz por docker-compose.yml, backend/go.mod y frontend/pubspec.yaml. Este skill vive en .agents/skills/mariposa-marketplace; desde esa carpeta la raíz es ../../.. . Las rutas de código siguientes son relativas a esa raíz.
- Para volver a ejecutar la app existente, sigue la preparación y validación de abajo. No reconstruyas código que ya funciona.
- Para reconstruirla en otro proyecto, lee [requisitos.md](references/requisitos.md) y usa el repositorio actual como referencia si está disponible. Trabaja en el destino solicitado; si falta el destino y crearlo podría sobrescribir trabajo, acláralo antes de escribir allí.
- Para retomar cambios, revisa git status y los archivos pertinentes. Lee README.md, backend/README.md y frontend/README.md según el alcance. El usuario puede limitar el trabajo a backend, frontend o documentación.
- Si se pide cumplimiento con un PDF, lee el PDF suministrado y contrasta cada requisito. Documentation/revision-final.md contiene una evaluación anterior, no sustituye el documento ni demuestra el estado de una nueva ejecución. No dependas de rutas personales a Downloads.

## Preparar y ejecutar

Requisitos: Docker con contenedores Linux, Docker Compose, Node.js 20 o superior para la suite (CI usa 22) y conexión para descargar dependencias. Go y Flutter se construyen en Docker; consulta sus versiones fijadas en los Dockerfile y lockfiles.

Desde la raíz:

~~~sh
docker compose up --build -d
~~~

- Tienda: http://localhost:3000
- API: http://localhost:8080
- Readiness: http://localhost:3000/api/ready
- Mailpit: http://localhost:8025

El proyecto de la demo se llama backend. Mongo necesita replica set para transacciones. El seed se ejecuta antes de API y worker; no debe sobrescribir saldos ni pedidos existentes. No uses down --volumes para simplemente volver a ejecutar la demo. Si se necesita un entorno vacío de pruebas, usa la suite aislada. Un reinicio destructivo del entorno persistente requiere que el usuario haya solicitado borrar esos datos.

## Invariantes que deben mantenerse

- Money usa enteros/unidades menores, moneda, escala y redondeo configurables; nunca float para cálculos monetarios. Las reglas de país provienen de datos.
- Motor: COMBO → SCALE sobre unidades elegibles → GIFT → base gravable → impuestos → total → crédito → snapshot. Escalas por tramos, no aplicar el porcentaje final a todas las unidades. Cada beneficio identifica promoción y unidades/cantidades.
- La confirmación usa la cotización persistida, sin recalcularla. Quote.total = Order.total aunque cambien precios o promociones.
- Pedido, crédito, idempotencia y outbox se guardan atómicamente. HTTP 201 solo después del commit; la orden debe poder consultarse inmediatamente. Replay no crea otro pedido ni vuelve a consumir crédito; misma clave y otro payload produce conflicto.
- API y worker son procesos independientes. El broker caído no impide guardar una orden. El worker publica/reconcilia el outbox y procesa dos destinos independientes; nunca confirma pedidos.
- ERP y PUSH son dummies. La deduplicación por destino, los reintentos y la DLQ no equivalen a exactly-once distribuido. Se exige idempotencia al receptor real. Un fallo de un destino no debe bloquear el otro ni borrar la orden.
- Mantén logs estructurados correlacionables por orden/evento/destino, sin credenciales. Distingue transacción persistida, publicación al broker y efecto completado.

## Interfaz y documentación

- Conserva beneficios visibles junto al resumen en escritorio y accesibles en móvil, textos sin recortes, controles fuera del teclado/insets e imágenes consistentes.
- El historial permite abrir el detalle completo del pedido e indica estados persistidos de ERP/PUSH. Mantén el aviso de que esa sección es solo visual para la prueba y no corresponde a un marketplace real; no conviertas un estado pendiente en éxito decorativo.
- Los productos y promociones de prueba viven en backend/testdata/seed.json. Sus ejemplos se documentan en el apartado separado de promociones de Documentation/index.html y Documentation/Readme.md. Actualiza código, datos y ejemplos juntos cuando cambien.
- Cada página de documentación empieza su numeración en 0; no continúa la numeración de otra. Conserva el índice enlazado de Documentation/index.html.
- El ADR publicado es de una página; incluye decisiones, alternativas, exclusiones y un riesgo con mitigación. No añadas detalles que lo hagan exceder ese límite.

## Verificar y entregar

Para preparar una entrega completa o repetir la prueba:

~~~sh
node scripts/test.mjs
~~~

Este comando construye imágenes, ejecuta formato/análisis Go y Dart, Go con integración/race, regresión de descarga de módulos, Flutter/goldens, Chrome y smoke de worker/DLQ/correo. Crea un proyecto Compose con volúmenes y puertos propios, recoge evidencia en reports/marketplace-test-*/ y elimina sus recursos al terminar. Puede instalar Chrome y sus dependencias; en Linux estas pueden requerir privilegios. No modifica los golden de referencia.

Para una modificación acotada ejecuta primero las comprobaciones pertinentes. No regeneres golden para ocultar fallos: verifica el cambio esperado y compara las imágenes. Los controles adicionales de seguridad y complejidad están en ci/README.md y .github/workflows/ci.yml. No confundas la suite funcional local con esos controles ni con CI remoto.

Si falla una descarga Go, conserva el máximo de tres intentos del script backend/scripts/download-go-modules.sh, TLS y checksums. Si persiste, reporta el error; no suprimas el fallo ni repitas indefinidamente.

Publica solo dentro del alcance autorizado. Si el usuario pide publicar y confirmar, identifica el SHA enviado y comprueba CI required y Pages de ese mismo SHA hasta su resultado final; investiga fallos en sus logs. Una ejecución antigua verde no valida cambios nuevos.

Entrega comandos, resultados realmente ejecutados y limitaciones pendientes. No reutilices cantidades históricas de tests como evidencia actual ni prometas un SLA o despliegue productivo.
