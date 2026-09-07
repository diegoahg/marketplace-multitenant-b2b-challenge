# MariposaMarket · Flutter frontend

Interfaz minimalista y adaptable para el marketplace B2B. Flutter Web 3.44.0, Material 3 y tipografía local. Paleta: primario `#123B5D`, secundario `#00A6A6`, acciones `#FFC629`, éxito `#22A06B`, fondo `#F5F7F8` y texto `#17212B`. Los CTA amarillos usan texto oscuro; las superficies secundarias usan tintes suaves. Los colores compartidos están en `lib/widgets.dart` y el tema Material en `lib/main.dart`. El catálogo presenta los dos productos comprables del seed; sus nombres e ilustraciones son de demostración. Los precios, promociones, impuestos, regalos y crédito siempre llegan desde Go.

## Ejecutar con Docker

Desde la raíz del repositorio:

```sh
docker compose up --build -d
```

Abrir **http://localhost:3000**. La API sigue disponible en `http://localhost:8080`. El proxy de nginx expone `/api` en el mismo origen de Flutter, sin cambiar CORS ni los contratos del backend. Mongo mantiene su volumen existente. El nombre del proyecto Compose sigue siendo `backend`, compatible con el entorno previo.

Si el backend ya está funcionando:

```sh
docker compose up --build --no-deps -d frontend
```

Si se recreó el emulador, el backend recupera sus recursos al recibir 404. La reconciliación repone eventos sin efectos completados después de un minuto; `/api/ready` refleja la recuperación. El emulador es volátil y Mongo conserva la outbox.

## Recorrido

1. Seleccionar Perú/PEN o Chile/CLP y agregar productos con `+`, `−` o el editor de cantidad.
2. Opcionalmente cargar una selección rápida: volumen, combo + volumen u obsequio.
3. Elegir crédito o contado y pulsar **Cotizar pedido**.
4. Revisar subtotal, descuentos, base gravable, impuestos, total, regalos, campañas y cálculo por línea.
5. Confirmar durante la vigencia de la cotización. Si el saldo cambió, se muestra el rechazo real del servidor.
6. Consultar el pedido confirmado en **Mis pedidos**, por ID o desde el historial de este navegador.

En móvil, el icono de bolsa del encabezado lleva directamente al pedido. Las cantidades admiten 0–1.000.000; cero elimina la línea. Para probar crédito insuficiente, usar una cantidad alta, cotizar a crédito y comprobar el bloqueo de confirmación.

Países disponibles: Perú (PEN), Chile (CLP), Colombia (COP), Ecuador (USD), Guatemala (GTQ) y Argentina (ARS). Los cuatro nuevos usan dos decimales y datos nominales de demo, sin conversión de divisas; su tasa de impuesto es 0 hasta configurar las reglas comerciales. COP, USD y ARS se muestran con código para distinguir monedas que comparten símbolo; GTQ usa Q. Las pruebas Chrome cotizan y confirman en cada país nuevo.

## Invariantes

- `Money` usa `BigInt` y formato decimal por cadenas. No hay cálculos monetarios con `double`, tampoco al compilar a JavaScript.
- El frontend no implementa otro motor de precios. Muestra el snapshot recibido y verifica que el total confirmado coincida exactamente en importe, moneda y escala.
- Cambiar país, cantidades o pago invalida la cotización. No se puede iniciar una confirmación vencida o no elegible a crédito.
- La clave UUID de idempotencia y el snapshot se guardan **antes** de enviar la confirmación. Timeout, fallo de red o 5xx conservan el mismo intento, incluso tras recargar. Durante ese estado se bloquea editar el carrito.
- Un replay pendiente puede ejecutarse después del vencimiento: el servidor distingue un pedido ya confirmado de una primera confirmación tardía.
- El botón evita solicitudes concurrentes dentro de la instancia de UI. Las garantías transaccionales entre pestañas, clientes y procesos siguen siendo responsabilidad del backend.
- El historial guarda hasta 50 pedidos en este navegador y filtra país, tenant y cliente. La consulta por ID usa siempre el scope seleccionado; el backend vuelve a validarlo.
- Almacenamiento local fallido antes de enviar: no se envía la confirmación. Si falla después de recibirla, se conserva el intento para recuperar el resultado.

## Pruebas y golden tests

Desde la raíz, con Mongo y Pub/Sub activos:

```sh
docker compose --profile frontend-test run --build --no-deps --rm frontend-tests
docker compose --profile test run --no-deps --rm tests
```

El primer comando ejecuta `flutter analyze` y `flutter test`; el segundo ejecuta las pruebas Go con `-race -tags=integration -count=1`. Las referencias visuales se comparan en Linux con la versión fijada de Flutter.

| Golden | Cobertura de frontend | Garantía comprobada en Go |
|---|---|---|
| GT01 | Snapshot real importado, descuento por tramos y total visible | Escala acumulativa |
| GT02 | Combo, excedentes, descuentos y total del snapshot | Combo antes de escala |
| GT03 | SKU, cantidad, campaña, valor e impuesto del regalo | Regla de obsequio |
| GT04 | Base gravable e impuesto recibidos | Impuesto después del descuento |
| GT05 | Bloqueo por crédito insuficiente y alternativa contado | Límite exacto del crédito |
| GT06 | Total preservado; rechazo de respuesta que no coincide | Snapshot pese a cambios de precios |
| GT07 | 100 intentos simultáneos en UI, una solicitud; replay tras recarga | Idempotencia con concurrencia real Mongo |
| GT08 | Rechazo de crédito sin inventar pedido; nueva cotización | Débito condicional y rollback |
| GT09 | Formato exacto, decimales, signos y más de 2^53 | HALF_UP / HALF_EVEN y aritmética exacta |
| GT10 | País/moneda/escala, invalidación e historial por scope | Configuración multi-país |
| GT11 | Estado pendiente ante 5xx y ausencia de éxito ficticio | Outbox recuperable y rechazo sin evento |
| GT12 | Replay no duplica el historial ni descarta la clave | Deduplicación, redelivery y efectos ERP/Push |

Los archivos `test/fixtures/quote_GT01.json`–`quote_GT05.json` se exportaron **del motor Go real** usando las mismas entradas de `backend/internal/pricing/engine_test.go`. Se comparan con las expectativas canónicas `GT01.json`–`GT05.json`, además de verificarse la presentación. El seed interactivo habilita varias promociones a la vez; por ejemplo, la selección de 12 unidades incluye regalos además del descuento, mientras GT01 aísla únicamente la escala. Los botones de selección no se presentan como una ejecución de la suite golden.

Hay ocho referencias visuales en `test/goldens`: catálogo de escritorio, tablet y móvil, cotización de escritorio y móvil, resumen con nombres largos a 768 px y dos diálogos con teclado simulado. Las pruebas visuales complementan los golden de negocio; no los reemplazan.

### Prevención de defectos de UX por componente

- **Texto completo:** `ProductGrid` calcula columnas según el ancho y la escala de texto, con altura natural por tarjeta. `AdaptivePair` apila etiquetas, importes y controles cuando falta espacio. Los nombres no usan elipsis ni límites de líneas. El editor de cantidad separa el nombre largo de la etiqueta breve del campo.
- **Controles accesibles:** `SafeAppScaffold` centraliza `SafeArea` y el ajuste por teclado; oculta la navegación de la app mientras el teclado está abierto. `SafeFormDialog` permite desplazar el contenido manteniendo accesibles las acciones. Las pantallas usan desplazamiento y abandonan la barra lateral si la altura disponible es insuficiente.
- **Imágenes consistentes:** `ProductMedia` impone a todas las tarjetas una ventana 4:3, margen de 16 y `BoxFit.contain`, centrado. Conserva las proporciones sin recortar; los distintivos se colocan fuera de la imagen.

`test/layout_test.dart` verifica nombres y campañas largos a 600, 768 y 1024 px con texto al 100 % y 150 %, márgenes del sistema y teclado mediante `viewPadding`/`viewInsets`, y geometría de imágenes horizontales y verticales. Estos componentes y sus pruebas se reutilizan en nuevas pantallas. Antes de aceptar cambios visuales, ejecutar la suite comparando los golden existentes; regenerarlos únicamente tras revisar el cambio intencional. Matriz y evidencias en [VALIDATION.md](VALIDATION.md).

### Navegador real

Con Chrome y Node instalados, desde `frontend`:

```sh
npm ci
npm run test:e2e
```

Playwright verifica la compilación release contra la API real: cotización con escala y regalos, combo + escala, cambio a Chile, crédito insuficiente y confirmación al contado cuya respuesta se pierde intencionalmente. Recarga la página, reintenta con la misma clave y comprueba que recibe el mismo pedido. Ese test crea **un pedido real de demo al contado**, sin consumir crédito. Capturas y trazas quedan en `test-results/` (ignorado por Git).

### Regenerar fixtures desde Go

Construir primero la imagen de pruebas del backend. Desde la raíz, en PowerShell:

```powershell
docker compose --profile test build tests
$frontendPath = (Resolve-Path frontend).Path
docker run --rm --mount "type=bind,source=$frontendPath,target=/frontend" backend-tests sh -c 'mkdir -p /src/cmd/frontend-fixtures && cp /frontend/tool/export_fixtures.go /src/cmd/frontend-fixtures/main.go && go run ./cmd/frontend-fixtures /frontend/test/fixtures'
```

El programa temporal se crea dentro del contenedor; no modifica el backend del repositorio. Los PNG deben actualizarse solo tras revisar visualmente un cambio deliberado, con `flutter test --update-goldens` usando la misma versión de Flutter/Linux.

## Estructura

```text
lib/models.dart            Money exacto, Scope, Quote, Order, catálogo visual
lib/api.dart               Contratos HTTP, errores y timeout
lib/store.dart             Persistencia local del intento e historial
lib/shop_controller.dart   Flujo de compra, invalidación y recuperación
lib/main.dart              Catálogo, carrito, desglose e historial adaptables
lib/widgets.dart           Componentes, editor de cantidades e ilustraciones
lib/adaptive_components.dart Reglas compartidas de texto, insets, diálogo y grid
test/                     Tests de estado, contrato, widgets y PNG golden
e2e/                      Pruebas Chrome con la API real
tool/export_fixtures.go    Exportación de snapshots desde el motor Go
Dockerfile / nginx.conf    Build reproducible y proxy de mismo origen
```

## Desarrollo local y límites

Con Flutter 3.44.0 instalado: `flutter pub get`, `flutter analyze`, `flutter test` y `flutter build web --release --no-web-resources-cdn`. Servir el build con el proxy nginx suministrado; el backend no habilita CORS para `flutter run -d chrome` directo a otro puerto. El proceso de compilación sigue la [documentación oficial de Flutter Web](https://docs.flutter.dev/deployment/web).

Esta entrega configura **Flutter Web**, con layout móvil; no añade proyectos Android/iOS. Tenant y cliente corresponden al seed de demostración, no a autenticación. El catálogo del backend no dispone de un endpoint de lectura: el frontend incluye sus SKU y nombres visuales, pero nunca un precio definitivo estático. No se muestran estados de ERP/Push porque la API no los expone. La deduplicación de efectos se valida con las pruebas de integración del backend. No se ofrecen pagos reales ni inventario. El almacenamiento del navegador no es un historial central ni una garantía de sincronización entre pestañas.
