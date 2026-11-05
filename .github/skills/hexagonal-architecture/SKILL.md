---
name: hexagonal-architecture
description: Reglas de arquitectura hexagonal (puertos y adaptadores) de este proyecto. Úsala al crear o modificar modelos, servicios, puertos o adaptadores.
---

# Arquitectura hexagonal

Estructura:
- `src/domain/models`: entidades y objetos de valor.
- `src/domain/services`: lógica de negocio / casos de uso.
- `src/domain/ports`: interfaces (puertos) que el dominio expone o necesita.
- `src/infrastructure/in`: adaptadores de entrada (API, CLI, etc.) que llaman al dominio.
- `src/infrastructure/out`: adaptadores de salida (BD, servicios externos) que implementan puertos.

Reglas:
1. El dominio NO importa nada de `infrastructure`.
2. La infraestructura depende del dominio, nunca al revés.
3. Toda interacción externa pasa por un puerto.
4. Los adaptadores se crean en pasos posteriores; no añadir funcionalidad sin pedirlo.
