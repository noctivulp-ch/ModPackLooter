# 10 · Proceso de pruebas, entrega y publicación

## Modpack de referencia

**DeseacedCraft BetaCerrada**: Forge, Minecraft 1.20.1. Es el caso real con
el que se valida el MVP. Ver [ADR-0008](decisiones/0008-ciclo-de-iteracion-con-modpack-real.md).

## Ciclo de iteración

```
 ┌─ 1. Desarrollo en el repo (rama claude/…)
 │      tests automáticos con fixtures pequeños
 ├─ 2. Release en GitHub (CI compila binarios para Windows/macOS/Linux)
 │      o el usuario clona y compila localmente
 ├─ 3. El usuario ejecuta:  modpacklooter build <ruta DeseacedCraft> --out site
 ├─ 4. El usuario revisa el sitio y devuelve:
 │        site.zip + mensaje de review (qué falla, qué falta, qué mejorar)
 │        + salida de `modpacklooter scan` / diagnósticos si hay errores
 └─ 5. Se analiza el zip, se convierte el feedback en tareas y se vuelve a 1
```

### Qué conviene incluir en cada review

- El `site.zip` generado (la app incluye `about/diagnostics.json` dentro).
- El comando exacto usado y la versión de la app (`modpacklooter --version`).
- Ejemplos concretos: “el objeto X debería salir en la estructura Y”.
- Capturas si algo se ve mal.

## Pruebas automáticas

| Nivel | Qué cubre |
|---|---|
| Unitarias | Dominio: probabilidades, resolución de tags, prioridad de recursos. |
| Fixtures por versión | JSON/NBT mínimos de 1.20.1, 1.21.x y 26.x (tomados de misode/mcmeta y de mods de licencia libre). |
| Contrato de puertos | La misma batería para todos los adaptadores de un puerto. |
| *Golden files* | Mini-modpack sintético → JSON del índice y HTML comparados con snapshots. |

El modpack real **no** se sube al repositorio (licencias de terceros).

## Entrega de la app

- **GitHub Releases**: binarios autocontenidos por SO/arquitectura,
  generados por `.github/workflows/app-release.yml` al crear una etiqueta
  `app/vX.Y.Z` (también se puede lanzar a mano para obtener binarios de prueba
  como artefactos del workflow).
- **CI** (`app-ci.yml`): formato, `go vet` y tests, incluidos los de integración con
  los datos vanilla reales de `misode/mcmeta`.
- Compilación local: un solo comando documentado en el README.

## Publicación del sitio generado

El sitio debe funcionar tanto en **GitHub Pages** como en **GitLab Pages**
(y en cualquier hosting estático):

- **Solo enlaces relativos**, porque Pages sirve el sitio bajo `/<repo>/`.
- Una carpeta con `index.html` por página (`/objetos/minecraft-diamond/`),
  sin depender de reescrituras del servidor.
- Archivo `.nojekyll` para que GitHub Pages no ignore carpetas con `_`.
- `modpacklooter init --ci github|gitlab` generará el workflow
  (`.github/workflows/pages.yml` o `.gitlab-ci.yml` con el job `pages`) para
  regenerar y publicar el sitio desde el repo del modpack.
