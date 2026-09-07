# Marketplace B2B · MariposaMarket

Backend Go con precios, promociones, crédito y pedidos; frontend Flutter Web minimalista y adaptable.

```sh
docker compose up --build -d
```

- **Tienda:** http://localhost:3000
- **API:** http://localhost:8080
- **Estado:** http://localhost:3000/api/ready
- **Correo de prueba (DLQ):** http://localhost:8025

Requiere Docker con contenedores Linux. El seed se carga automáticamente. Se conserva el volumen Mongo del proyecto `backend`.

```sh
docker compose --profile frontend-test run --build --no-deps --rm frontend-tests
docker compose --profile test run --build --no-deps --rm tests
```

Detalles de ejecución, arquitectura, pruebas y límites: [frontend](frontend/README.md), [backend](backend/README.md), [documentación del desafío](Documentation/Readme.md). [ADR de una página](Documentation/adr.html) · [PDF](Documentation/ADR.pdf).

![Arquitectura actual](Documentation/images/arquitectura.svg)

[Descargar arquitectura en PNG](Documentation/images/arquitectura.png).

[Revisión final contra el PDF](Documentation/revision-final.md) · [Colección de Postman y entorno](backend/contracts/postman/README.md) · [Worker, reintentos y DLQ](backend/worker/README.md).

Integración continua: [workflow de pruebas, sintaxis, complejidad y seguridad](.github/workflows/ci.yml). Consulta los controles y la evidencia local e histórica en [ci/README.md](ci/README.md). El estado de una ejecución anterior no acredita los cambios todavía sin publicar.
