---
name: hexagonal-architecture
description: Reglas de arquitectura hexagonal para esta aplicación Go. Úsala al crear o modificar modelos, servicios, puertos, adaptadores y composición de dependencias.
---

# Arquitectura hexagonal

## Estructura actual

- `src/domain/models`: entidades y objetos de valor, sin detalles de HTTP ni persistencia.
- `src/domain/services`: reglas de negocio y orquestación de casos de uso; distinguir ambos roles sin añadir capas innecesarias.
- `src/domain/ports`: interfaces y contratos que necesita o expone el núcleo.
- `src/infrastructure/in`: adaptadores de entrada (por ejemplo, HTTP o CLI).
- `src/infrastructure/out`: adaptadores de salida (por ejemplo, bases de datos o servicios externos).
- `src/infrastructure/out/db/internal/dbo`: representaciones propias de persistencia cuando se justifique.
- `src/infrastructure/out/db/internal/mapper`: conversiones de persistencia cuando se justifiquen.
- `src/cmd/server`: punto de arranque y lugar apropiado para conectar implementaciones concretas con puertos, directamente o mediante funciones de composición.

## Dependencias y contratos

1. `src/domain` no importa paquetes de `src/infrastructure`.
2. La infraestructura depende de contratos del dominio y no al revés.
3. Las dependencias externas que utilizan los casos de uso deben estar mediadas por puertos cuando ello aísle una decisión de infraestructura; no crear una interfaz por cada función ni por cada operación interna de un adaptador.
4. Mantener los puertos expresados con tipos del dominio y errores apropiados al contrato, no con DTO HTTP, DBO, SQL ni tipos específicos del motor de base de datos.
5. Los adaptadores de entrada traducen protocolos y errores de transporte; los de salida implementan contratos y traducen representaciones/errores de infraestructura cuando corresponda.
6. Conectar las implementaciones concretas en la composición de la aplicación, no dentro de las entidades ni los casos de uso.

## Persistencia y representaciones

- SQLite es la implementación de desarrollo actual; el motor de producción todavía no está decidido. No asumir ni incorporar PostgreSQL sin una solicitud expresa.
- Diseñar contratos de persistencia independientes del motor; evitar exponer SQL o detalles específicos de SQLite por los puertos.
- DTO de transporte y DBO de persistencia son responsabilidad de los adaptadores y no atraviesan los puertos.
- Convertir explícitamente las representaciones cuando haya diferencias o se requiera aislamiento. Crear DTO, DBO o mappers solo si ofrecen un beneficio concreto; evitar duplicaciones sin valor.
- Respetar las diferencias reales entre motores (tipos, restricciones, transacciones y consultas). No prometer intercambiabilidad automática entre SQLite y PostgreSQL.
- Definir el límite transaccional según el caso de uso sin introducir dependencias de infraestructura en el dominio.

## Convenciones Go

- Usar interfaces pequeñas centradas en lo que necesita el consumidor; preferir construcciones idiomáticas de Go.
- Respetar las reglas de importación de `internal`; los paquetes bajo `src/infrastructure/out/db/internal` solo son accesibles a paquetes autorizados por el compilador de Go.
- Propagar `context.Context` cuando exista una operación cancelable, con deadline o I/O; evitar guardarlo como campo permanente de entidades.
- Propagar y envolver errores preservando su semántica; no filtrar errores de motor o mensajes SQL directamente en las respuestas externas.
- No modificar nombres ni estructura pública sin necesidad; preservar contratos existentes salvo cambio expresamente requerido.

## Alcance del trabajo

- Implementar únicamente lo solicitado y los cambios indispensables para que funcione de extremo a extremo, incluidos los adaptadores necesarios cuando el requerimiento los implique.
- No crear capas, puertos, DTO, DBO, mappers o abstracciones adicionales por anticipación.
- Coordinar con la skill `testing`: al cambiar dominio, puertos o adaptadores, actualizar las pruebas correspondientes y verificar los contratos afectados.
- No modificar infraestructura de producción ni decidir el motor de base de datos de producción sin una instrucción explícita.
