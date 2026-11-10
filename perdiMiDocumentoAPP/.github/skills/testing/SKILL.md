---
name: testing
description: Guía de testing del proyecto. Úsala al escribir o ejecutar tests de dominio y adaptadores.
---

# Testing

- Tests en `tests/`, reflejando la estructura de `src/`.
- Dominio: tests unitarios puros, sin I/O; los puertos se sustituyen por dobles (fakes/mocks).
- Adaptadores IN/OUT: tests de integración contra el puerto correspondiente.
- Ejecutar los tests dentro de Docker: `docker compose -f docker-compose.dev.yml run --rm app <comando de test>`.
- Cada funcionalidad nueva debe incluir sus tests.
