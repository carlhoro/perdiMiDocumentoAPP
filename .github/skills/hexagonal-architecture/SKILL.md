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

Límites y representaciones:
- Los puertos usan tipos y contratos del dominio; los adaptadores pueden importarlos e implementarlos sin invertir la dirección de las dependencias.
- Mantener los modelos del dominio independientes de formatos HTTP y detalles de persistencia. Los DTO de transporte y los DBO de persistencia pertenecen a sus adaptadores y no atraviesan los puertos.
- Convertir explícitamente entre representaciones en el límite del adaptador, conservando el contrato público, los errores y las reglas de negocio.
- Crear DTO o DBO cuando exista una representación o necesidad de aislamiento concreta; evitar duplicar tipos sin beneficio.
