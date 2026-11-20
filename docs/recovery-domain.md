# Núcleo de recuperación

El paquete `src/application/recovery` implementa los casos de uso del nuevo dominio.
Está aislado del HTTP, SQLite y autenticación heredados: **las garantías siguientes
todavía no protegen la aplicación que arranca `src/cmd/server`**.

## Composición

Construir `Service` con `Transactions`, `Identity`, `Actors` y `Clock` no nulos.
`Ownership` ausente bloquea seguimientos y recuperación; `Capacity` ausente limita
a un seguimiento. Los plazos predeterminados son 30 días de vigencia y 24 horas
para duplicados del mismo reportante/documento/ubicación. No modificar el reloj
ni la configuración durante operaciones concurrentes.

El actor es el ID local resuelto por el adaptador de autenticación, nunca un ID
confiado desde el cuerpo de una solicitud. `ActorAuthorizer` comprueba su vigencia.
Las entidades referencian la cuenta existente; no se crea otro registro de usuarios.

## Contratos de adaptación

- `IdentityProtector.Lookup` recibe una identidad transitoria normalizada CO/CC,
  conserva ceros iniciales y devuelve referencias HMAC versionadas para todas las
  claves legibles. No usar hashes simples ni persistir el original. La referencia
  actual ocupa la primera posición. Un adaptador real debe definir serialización
  inequívoca, separación de propósito y administración segura de claves.
- `ResolveDocument` resuelve cualquier versión al mismo documento canónico,
  incorpora la referencia vigente y rechaza referencias que apunten a identidades
  incompatibles. No crear documentos en búsquedas sin coincidencias.
- `OwnershipVerifier` verifica la pareja exacta usuario/documento. Un login OIDC,
  nombre o email no acredita el número. El adaptador futuro valida emisor confiable,
  audiencia, evidencia, vigencia y revocación antes de exponer la decisión. No
  almacenar tokens ni pruebas fotográficas en las entidades.
- Las verificaciones dentro de una transacción deben consultar evidencia confiable
  local y coherente con su aislamiento; una futura llamada remota no debe mantener
  abierta la transacción. Su sincronización y revocación requieren diseño de
  integración antes de producción.
- `UnitOfWork.Within` debe revertir todas las escrituras ante error y serializar
  operaciones que compitan por cuota, documento, reporte o coincidencia. Repositorios
  entregan valores independientes: modificar una entidad no modifica la persistencia.
- Imponer unicidad de referencias protegidas, usuario/documento, actor/clave de
  operación y usuario/reporte/clase de notificación. Insertar una notificación ya
  existente es un éxito idempotente. `SaveReport` compara versiones al actualizar.
- Auditoría de éxito, registro y notificaciones comparten transacción. Los rechazos
  de entrada se auditan separadamente sin cuerpos originales. Un fallo de auditoría
  se propaga; no se oculta como validación exitosa.

## Comportamiento

La búsqueda requiere autenticación antes de normalizar o consultar y devuelve solo
ubicación permitida, estado y vencimiento. Una ubicación privada o desconocida omite
nombre y dirección precisa. Las ubicaciones públicas son declaraciones sin acreditar,
no pruebas de custodia; no se publican nombres ni números de titulares.

Los reportes se crean sin poseer físicamente el documento. La vigencia se evalúa al
leer y notificar aunque no exista un trabajo programado. Los estados terminales no
se reabren. `PotentialContradiction` permite detectar múltiples ubicaciones vigentes
sin eliminar reportes ni afirmar traslado. Confirmar custodia está deshabilitado.

Observer se coordina síncronamente en aplicación: registro de reporte y activación
de seguimiento producen coincidencias en ambos órdenes. La revocación suspende el
seguimiento cuando se evalúa para producir o leer notificaciones. La lectura actualiza
disponibilidad y privacidad; no devuelve cuerpos personales persistidos.

Inicialmente solo se admite una cédula activa distinta por usuario. Aumentar el cupo
no permite otros titulares ni eludir verificación. Un cambio de cupo no cancela una
activación existente; detenerla libera capacidad. La evidencia de un nuevo documento
debe resolver la sustitución antes de habilitarlo.

## Corte y migración futuros

No conectar estos casos de uso a HTTP hasta disponer de adaptadores reales y pruebas
de contrato. No reutilizar DTO heredados ni inferir CO/CC de solicitudes incompletas.
Un acceso anónimo debe rechazarse sin revelar siquiera existencia o ubicación.

Migrar primero una copia aislada: inventariar tipos y duplicados sin datos en logs;
apartar país/tipo ambiguos; proteger identidades; convertir filas antiguas en reportes
con procedencia explícita; tratar direcciones desconocidas como restringidas. La fecha
antigua de registro no acredita un avistamiento. Las suscripciones antiguas se suspenden
hasta acreditar titularidad; las notificaciones antiguas no se publican sin autorización.
Validar recuentos, idempotencia, rollback y reinicio antes del corte. No usar escritura
doble ni conservar caminos heredados que evadan autorización tras activarlo.

Sin originales no puede reconstruirse un HMAC nuevo a partir del anterior. Una rotación
total requiere reingreso, vinculación progresiva o una estrategia adicional. La eliminación
de datos personales del almacenamiento activo debe tratar respaldos por separado.

## Verificación

Las pruebas nuevas de dominio son deterministas; las transversales usan un almacén
transaccional y proveedores falsos exclusivos de pruebas. El prefijo `TEST-ONLY` no
es criptografía y no debe convertirse en un adaptador real. Estos dobles no prueban
aislamiento de SQLite: cada adaptador futuro debe demostrar rollback, concurrencia,
unicidad, persistencia y migración con una base independiente bajo `t.TempDir()`.

Ejecutar `go test ./...` o el comando Docker documentado en README. Para verificar
concurrencia con detector de carreras, usar `go test -race ./...` solo en un entorno
con CGO y compilador C disponibles.
