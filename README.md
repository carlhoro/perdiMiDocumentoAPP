# Perdí mi documento

Aplicación web en Go para registrar documentos encontrados, buscar por número de identificación y recibir notificaciones dentro de la aplicación.

## Desarrollo con Docker

Requiere Docker y Docker Compose. Desde esta carpeta:

```sh
docker compose -f docker-compose.dev.yml up --build
```

Abrir http://localhost:8080. Crear una cuenta e iniciar sesión. La información persiste en el volumen `dev_data` de SQLite, incluso al detener los contenedores. El código se monta en el contenedor; reiniciar el servicio para aplicar cambios.

Para ejecutar las pruebas:

```sh
docker compose -f docker-compose.dev.yml run --rm app go test ./...
```

Alternativamente, con Go 1.24 instalado: `go run ./src/cmd/server` y `go test ./...`. Por defecto la base se crea en `data/documentos.db`; `DB_PATH` cambia su ruta y `HTTP_ADDR` cambia la dirección del servidor.

## Uso

Un usuario autenticado puede registrar el número, tipo, nombre del titular y dirección donde se encuentra un documento. La búsqueda es exacta por número, conservando ceros iniciales. La suscripción es por número; cuando se registra un documento coincidente se crea una notificación persistente para cada usuario suscrito. Al suscribirse a un número ya registrado también se generan notificaciones; repetir la suscripción no las duplica. La interfaz las consulta cada 15 segundos o al pulsar Actualizar. No se envían correos ni mensajes externos.

El número se compara literalmente después de quitar espacios en los extremos; no se valida contra registros oficiales. **Esta versión no verifica la titularidad**: cualquier usuario registrado puede consultar o suscribirse a cualquier número y ver el nombre y dirección. No debe utilizarse con datos personales reales ni exponerse públicamente sin implementar verificación de identidad, controles de acceso, protección contra abuso y una política de privacidad.

## API JSON

Las rutas protegidas requieren la cookie de sesión obtenida en el ingreso. Las solicitudes POST usan `Content-Type: application/json`.

| Método y ruta | Cuerpo / consulta | Resultado |
|---|---|---|
| POST `/api/registro` | `{"usuario":"ana","clave":"claveSegura"}` | Crea una cuenta |
| POST `/api/ingreso` | `{"usuario":"ana","clave":"claveSegura"}` | Inicia sesión |
| POST `/api/salir` | `{}` | Cierra sesión |
| GET `/api/sesion` | — | Usuario de la sesión |
| POST `/api/documentos` | `{"numero":"00123","tipo":"cédula","nombre_titular":"Ana","direccion":"Oficina central"}` | Registra documento |
| GET `/api/documentos?numero=00123` | — | Lista de coincidencias |
| POST `/api/suscripciones` | `{"numero":"00123"}` | Suscribe al usuario |
| GET `/api/notificaciones` | — | Notificaciones del usuario |

## Diseño y datos

El dominio define modelos, puertos y servicios; los adaptadores HTTP y SQLite implementan las entradas y salidas. `DocumentFactory.Create` crea y valida los objetos de documento. `DocumentProxy` intercepta la creación y audita los rechazos; el repositorio registra las operaciones exitosas en `logs` con usuario, número, documento y fecha. El registro, su log y las notificaciones se guardan en la misma transacción. El sujeto notifica a los observadores persistidos por número, de manera que las suscripciones sobreviven a reinicios. La conexión principal se inicializa mediante singleton.

El adaptador SQLite está organizado por responsabilidad en `src/infrastructure/out/db`; sus representaciones de filas se convierten a modelos de dominio en el límite del repositorio. El adaptador HTTP utiliza DTO para sus respuestas JSON, mientras que los puertos siguen utilizando modelos del dominio.

Las contraseñas se guardan con bcrypt; los tokens de sesión se guardan como hashes y caducan a las 24 horas. Las fechas en SQLite son Unix UTC. `logs.operation` distingue `registro` y `registro_rechazado`.

## Antes de publicar

No añadir credenciales, datos reales ni copias de la base al repositorio o a la imagen. Los archivos `.env*`, las bases SQLite y sus archivos auxiliares, los registros y formatos comunes de claves privadas se excluyen de Git y del contexto de Docker. La aplicación no necesita credenciales preconfiguradas ni un archivo `.env` para desarrollo.

Estas reglas no cubren todos los nombres o formatos posibles: si `DB_PATH` apunta a otro nombre, carpeta o extensión, excluirlo expresamente junto con sus copias y archivos auxiliares. Revisar también el contenido que se vaya a agregar. `.gitignore` no deja de versionar archivos ya seguidos ni elimina información del historial; ante una credencial expuesta, revocarla o rotarla y tratar el historial por separado.

## Imagen de producción

`docker compose -f docker-compose.prod.yml up --build -d` ejecuta la imagen compilada con su propio volumen `prod_data`. Es una base de despliegue, no una configuración lista para exposición pública. Para HTTPS mediante un proxy inverso, configurar `COOKIE_SECURE=true`; no hacerlo en HTTP local porque el navegador no enviará la cookie. Proteger y respaldar el volumen de base de datos.
