# perdiMiDocumentoAPP

Arquitectura hexagonal. Estructura inicial:

```
src/
  domain/{models,services,ports}
  infrastructure/{in,out}   # adaptadores de entrada y salida
tests/
```

Docker:
- Desarrollo: `docker compose -f docker-compose.dev.yml up --build`
- Producción: `docker compose -f docker-compose.prod.yml up --build -d`

Skills en `.github/skills/` (arquitectura y testing).
