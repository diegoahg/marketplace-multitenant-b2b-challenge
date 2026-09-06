# Validación del frontend

Cierre: **6 de septiembre de 2026**. Flutter Web 3.44.0 compilado en Docker Linux; navegador Chrome headless mediante Playwright 1.63.0.

| Comprobación | Resultado |
|---|---|
| Build release y nginx con proxy `/api` | PASS; frontend levantado en http://localhost:3000 |
| `flutter analyze` | PASS, sin incidencias |
| `flutter test` | **42 pruebas aprobadas**; se compararon los ocho PNG golden sin regenerarlos |
| `npm run test:e2e` | **4 pruebas aprobadas** |
| Consola del navegador | Sin errores de Flutter ni excepciones JS en los cuatro recorridos; solo se permite el error de red provocado deliberadamente en la prueba de respuesta perdida |
| `GET /api/ready` | `ready` |

Las pruebas de Go con `-race -tags=integration -count=1` también pasaron durante la implementación previa al corte. No se modificó código de negocio del backend. La matriz GT01–GT12, sus límites y la distinción entre golden de negocio y golden visual están en [README.md](README.md#pruebas-y-golden-tests).

## Recorridos reales verificados

1. Perú: 12 unidades, total **13.63 PEN**, descuento de 0.45 PEN y dos regalos. Confirmación al contado, pérdida intencional de la respuesta después del commit, recarga, replay con la misma clave y recuperación del mismo pedido. Consulta desde el historial y por identificador.
2. Combo + escala: total **10.56 PEN**, descuento 1.05 PEN, base 8.95 PEN e impuesto 1.61 PEN; escala únicamente sobre la unidad excedente elegible.
3. Móvil de 390 × 844: cambio a Chile, cotización de 12 unidades por **13,745 CLP**, edición de cantidad a 1,000,000, crédito insuficiente y confirmación deshabilitada.
4. Pantalla mediana: abrir el editor de cantidad a 768 × 1024 y reducir la altura a 460 px; el botón Aplicar permanece dentro del viewport y permite guardar 18 unidades.

## Criterios de la pantalla de resumen

La pantalla consume la cotización de Go y muestra las líneas del carrito, subtotal, descuentos, base gravable, impuestos y total. El detalle identifica el tipo y la campaña de cada descuento, la cantidad y campaña de los obsequios, su impuesto y el cálculo por producto.

| Defecto que se previene | Componente compartido | Verificación |
|---|---|---|
| Texto cortado | `ProductGrid`, `AdaptivePair`, alturas naturales y nombres separados de etiquetas de entrada | Seis escenarios de nombres y campañas largos: anchos 600/768/1024 px × texto 100/150 %. Se comprueban desbordamientos y límites de los glifos, antes y después de expandir el cálculo. |
| Controles tapados | `SafeAppScaffold`, `SafeFormDialog`, contenido desplazable | Barra de sistema de 48 px y margen superior de 24 px. Diálogo a 390 × 844 y 768 × 600 con teclado de 280 px; Aplicar visible y operativo. Confirmación con teclado de 320 px y consulta de pedido con teclado de 280 px. |
| Escalado desigual de imágenes | `ProductMedia` con ventana 4:3, margen 16 y `BoxFit.contain` centrado | Dos imágenes de proporciones opuestas, 400 × 100 y 100 × 400: ventanas iguales, escala horizontal igual a vertical y contenido dentro del área útil. |

Los insets del sistema y del teclado se simulan en Flutter; Chrome añade una prueba real de interacción al reducir la altura disponible. No se ha probado un teclado físico ni una compilación nativa Android/iOS.

Capturas revisadas: [resumen con nombres largos a 768 px](test/goldens/summary_medium_long_names.png), [diálogo móvil](test/goldens/keyboard_390.png) y [diálogo mediano](test/goldens/keyboard_768.png). El resumen capturado corresponde a la tarjeta de totales; las aserciones de texto también cubren el detalle de campañas, obsequios y líneas.

La prevención se aplica al reutilizar los componentes de [adaptive_components.dart](lib/adaptive_components.dart) y ejecutar sus pruebas con cada cambio. Las referencias golden se actualizaron por el cambio deliberado de layout y después se compararon en una ejecución independiente sin `--update-goldens`.

Las confirmaciones de navegador crean pedidos de demostración al contado; no consumen crédito. Los escenarios de concurrencia entre procesos y de Pub/Sub/ERP/Push se verifican en las pruebas Go, no mediante estados simulados en la tienda.

## Correcciones al cerrar la revisión

- Se aplicó la paleta solicitada: azul `#123B5D`, turquesa `#00A6A6`, CTA amarillo `#FFC629` con texto `#17212B`, éxito `#22A06B` y fondo `#F5F7F8`. También se actualizaron las ilustraciones y las ocho referencias visuales; el tema y los componentes comparten los colores.
- El identificador del historial se expone como texto accesible y conserva el botón de copia.
- El `ExpansionTile` del detalle de precios tiene su propia `PageStorageKey`. Antes heredaba la clave del scroll y podía intentar leer un número como booleano al cotizar después de desplazarse en móvil. Se añadió una prueba que reproduce ese recorrido y abre el detalle; también se comprueba la consola en todos los tests de Chrome.

Comandos reproducibles desde la raíz:

```sh
docker compose up --build --no-deps -d frontend
docker compose --profile frontend-test run --build --no-deps --rm frontend-tests
```

Desde `frontend`: `npm ci` y `npm run test:e2e`. Requiere backend disponible y Chrome instalado. Las capturas de navegador quedan en `test-results/`; las ocho referencias visuales de la entrega están en `test/goldens/`.

Los cambios están guardados en el workspace y todavía no tienen commit. La entrega configura Flutter Web adaptable; no incluye builds nativos Android/iOS ni autenticación de producción.
