---
name: testing
description: Reglas de pruebas para la aplicación Go. Úsala al crear o modificar funcionalidades, corregir defectos, implementar adaptadores o ejecutar pruebas.
---

# Testing

## Alcance y organización

- Usar el paquete estándar `testing` de Go y archivos `*_test.go`.
- Preferir tests junto a los paquetes Go que prueban (`src/...`) cuando facilite acceso a contratos o comportamiento interno. Reservar `tests/` para escenarios transversales, de extremo a extremo o de integración entre paquetes. No mover pruebas existentes sin necesidad.
- Mantener el comportamiento observable como objetivo principal: evitar pruebas acopladas a detalles internos que puedan cambiar sin alterar los contratos.
- Cada funcionalidad nueva o corrección de defecto debe incluir pruebas relevantes; para bugs, agregar una prueba que reproduzca la regresión cuando sea viable.

## Tipos de pruebas

- **Dominio (`src/domain`)**: pruebas unitarias rápidas y deterministas, sin archivos, red ni base de datos. Sustituir las dependencias externas definidas por puertos con fakes, stubs o mocks según la necesidad, prefiriendo dobles simples.
- **Adaptadores de entrada (`src/infrastructure/in`)**: probar la conversión de solicitudes y respuestas, validaciones de transporte y traducción de errores; usar un caso de uso doble cuando corresponda.
- **Adaptadores de salida (`src/infrastructure/out`)**: probar que implementan el contrato de los puertos. Utilizar integraciones reales controladas cuando sea necesario (por ejemplo, base de datos temporal para SQLite).
- Cubrir casos exitosos, errores relevantes y condiciones límite, sin imponer porcentajes de cobertura arbitrarios.

## Aislamiento de datos

- Nunca ejecutar pruebas destructivas contra la base de datos de desarrollo (`DB_PATH=/app/data/documentos.db`) ni contra producción.
- Para pruebas SQLite, crear una base de datos independiente por prueba o suite, preferiblemente bajo `t.TempDir()`; cerrar conexiones y limpiar recursos.
- No asumir que SQLite y PostgreSQL tienen el mismo comportamiento. Si se incorpora otro motor, ejecutar pruebas de contrato/integración contra ese motor antes de utilizarlo en producción.
- Las pruebas deben ser repetibles y no depender del orden de ejecución.

## Ejecución

- Ejecutar la suite en el contenedor de desarrollo:
  `docker compose -f docker-compose.dev.yml run --rm app go test ./...`
- Para consultar cobertura cuando sea útil:
  `docker compose -f docker-compose.dev.yml run --rm app go test -cover ./...`
- Para investigar concurrencia cuando sea pertinente, ejecutar `go test -race ./...` dentro del contenedor **solo si** dispone de CGO habilitado y compilador C; no convertirlo en requisito automático si el entorno no lo soporta.
- Antes de ejecutar pruebas de integración, verificar que sus dependencias y datos de prueba estén aislados. No modificar la configuración de producción para hacer funcionar tests.

## Criterio de finalización

- Ejecutar las pruebas afectadas y, cuando resulte razonable, `go test ./...`.
- Informar comandos ejecutados, resultados, pruebas no ejecutadas y motivo; nunca afirmar que una prueba pasó sin haberla ejecutado.
- Ante fallos, investigar la causa; no eliminar pruebas ni relajar aserciones solo para obtener resultados verdes.

