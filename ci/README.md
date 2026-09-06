# CI de MariposaMarket

El workflow [ci.yml](../.github/workflows/ci.yml) corre en cada push, pull request, ejecución manual y semanalmente los lunes a las 07:23 UTC. La ejecución programada vuelve a consultar vulnerabilidades aunque no cambie el código.

| Job | Controles y condiciones de fallo |
|---|---|
| Go syntax and tests | `gofmt`, `go vet -tags=integration`, todas las pruebas Go con `-race -tags=integration -count=1`, Mongo replica set y Pub/Sub emulator. |
| Flutter syntax and golden tests | Formato Dart, `flutter analyze` y `flutter test` en Flutter 3.44.0/Linux. Compara los ocho golden; nunca los regenera. |
| Go and Dart complexity | Gocyclo 0.6.0 y Dart Code Linter 4.3.0. Complejidad ciclomática máxima 20; Go excluye archivos de pruebas y Dart analiza `lib`. Cualquier exceso hace fallar el job. |
| Security and workflow validation | Actionlint 1.7.7 para YAML y expresiones de Actions; Gosec 2.29.0 para código Go con severidad y confianza al menos medias; Trivy 0.74.0 para secretos y vulnerabilidades HIGH/CRITICAL en dependencias Go, Dart y npm, incluidas dependencias de desarrollo. |
| Chrome end-to-end in Docker | Compila la aplicación, espera `/api/ready`, verifica sintaxis JS y ejecuta los cuatro recorridos Playwright con Chrome y API real. |
| CI required | Falla si cualquier job anterior falla, se cancela o se omite. |

El workflow usa `contents: read`, acciones fijadas por SHA, Trivy fijado por digest y no persiste credenciales del checkout. No necesita credenciales cloud, claves de scanners ni servicios de pago. Los jobs de integración y navegador tienen entornos independientes y eliminan sus contenedores y volúmenes al terminar. Los informes de complejidad/seguridad y las evidencias de pruebas fallidas se conservan como artefactos durante siete días. Bash utiliza `pipefail`, por lo que `tee` no oculta fallos de los scanners.

Para impedir merges con controles fallidos, seleccionar **CI required** como comprobación obligatoria en las reglas de protección de la rama de GitHub. El archivo crea la comprobación; no modifica las reglas del repositorio. El workflow de Pages existente sigue separado.

## Validación local del 6 de septiembre de 2026

Los comandos se ejecutaron en Docker sobre el workspace; el workflow todavía no se ha ejecutado en GitHub Actions.

- Actionlint: sin errores de sintaxis/expresiones en los workflows.
- Go: pruebas unitarias, golden e integración con detector de carreras aprobadas; `go vet` aprobado. Se normalizó con `gofmt` el formato de tres archivos de pruebas que no cumplían el nuevo control.
- Flutter: análisis sin incidencias, formato sin cambios y 42 pruebas aprobadas, incluyendo comparación de los ocho golden.
- Chrome: cuatro recorridos aprobados; sintaxis de configuración y pruebas JS válida.
- Dart Code Linter: sin excesos de complejidad en `lib`. Se verificó además que una función temporal de complejidad 31 produce salida 2 y bloquea el control.
- Gosec: cero hallazgos con los umbrales configurados, sobre 19 archivos Go.
- Gocyclo: **FAIL esperado por deuda existente**, detallada abajo.
- Trivy: **FAIL esperado por dependencias existentes**, detalladas abajo.

### Hallazgos que bloquean CI

| Función Go | Complejidad | Límite |
|---|---:|---:|
| `pricing.Calculate` (`backend/internal/pricing/engine.go`) | 98 | 20 |
| `messaging.(*Worker).Handle` (`backend/internal/messaging/worker.go`) | 23 | 20 |
| `domain.ParseMoney` (`backend/internal/domain/money.go`) | 23 | 20 |

Trivy detectó 13 vulnerabilidades HIGH/CRITICAL en `backend/go.mod`: 12 altas y una crítica, asociadas a `golang.org/x/crypto v0.33.0` y `golang.org/x/text v0.22.0`. El informe señala versiones corregidas hasta `x/crypto v0.55.0` y `x/text v0.39.0`. Es un análisis de dependencias, no una prueba de que todas las rutas vulnerables sean alcanzables desde esta API. Los informes del job incluyen identificadores, versiones y referencias para la corrección.

No se aumentaron umbrales ni se añadieron excepciones para ocultar estos hallazgos. Antes de obtener un CI verde hay que refactorizar las tres funciones y actualizar las dependencias afectadas, comprobando compatibilidad y repitiendo las pruebas.

## Reproducir los controles

Desde la raíz, en Bash (las monturas usan `$PWD`; en PowerShell puede usarse la ruta absoluta):

```bash
docker build -t marketplace-ci-tools -f ci/Dockerfile .
docker run --rm -v "$PWD:/repo:ro" marketplace-ci-tools actionlint -color
docker run --rm -v "$PWD:/repo:ro" marketplace-ci-tools gocyclo -over 20 -ignore '_test.go' backend
docker run --rm -v "$PWD:/repo:ro" -w /repo/backend marketplace-ci-tools gosec -severity medium -confidence medium ./...
docker run --rm -v "$PWD:/repo:ro" -w /repo aquasec/trivy:0.74.0 fs --config ci/trivy.yaml --include-dev-deps .
docker compose --profile frontend-test run --build --no-deps --rm frontend-tests
docker compose --profile test run --build --rm tests
```

El comando completo de complejidad Dart está en el workflow. Las pruebas de navegador se ejecutan con `npm ci` y `npm run test:e2e` desde `frontend`, después de levantar la aplicación.

Referencias de las herramientas: [Actionlint](https://github.com/rhysd/actionlint), [Gosec](https://github.com/securego/gosec), [Dart Code Linter](https://pub.dev/packages/dart_code_linter) y [cobertura Dart de Trivy](https://trivy.dev/docs/latest/coverage/language/dart/). Estas revisiones automatizadas cubren las reglas configuradas; no equivalen a una auditoría manual ni a pruebas de penetración.
