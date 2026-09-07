# Marketplace B2B · MariposaMarket

Backend Go con precios, promociones, crédito y pedidos; frontend Flutter Web minimalista y adaptable.

```sh
docker compose up --build -d
```

- **Tienda:** http://localhost:3000
- **API:** http://localhost:8080
- **Estado:** http://localhost:3000/api/ready

Requiere Docker con contenedores Linux. El seed se carga automáticamente. Se conserva el volumen Mongo del proyecto `backend`.

```sh
docker compose --profile frontend-test run --build --no-deps --rm frontend-tests
docker compose --profile test run --build --no-deps --rm tests
```

Detalles de ejecución, arquitectura, pruebas y límites: [frontend](frontend/README.md), [backend](backend/README.md), [documentación del desafío](Documentation/Readme.md). [ADR de una página](Documentation/adr.html) · [PDF](Documentation/ADR.pdf).

Integración continua: [workflow de pruebas, sintaxis, complejidad y seguridad](.github/workflows/ci.yml). Consulta los controles, la validación local y los hallazgos que bloquean el CI en [ci/README.md](ci/README.md).
