# ADR-0013: Mundo modelo en lugar de un menú de creación de mundos

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

La generación depende de opciones que se eligen al crear el mundo y que los mods amplían
(perfil de Lost Cities, presets, biomas, datapacks). Replicar el menú de creación de mundos
de Minecraft con todas las opciones de cada mod sería enorme y frágil.

## Decisión

- `--world` acepta un mundo ya creado con el modpack (carpeta con `level.dat` o nombre en
  `saves/`). La app lee de él dimensiones, listas de biomas, datapacks activos y
  `serverconfig/`.
- Sin mundo se usan los valores que el pack asigna a mundos nuevos (`defaultconfigs/`) y,
  después, `config/`.
- Las configs de servidor se leen siempre a través de un único método (`ServerConfig`) con
  ese orden de prioridad.

## Consecuencias

- Un creador de modpacks puede generar el sitio "tal como lo verá el jugador" creando un
  mundo de prueba.
- Las estructuras y biomas que no existen en el mundo se detectan con seguridad.
- Los datos del mundo del usuario no se suben al repositorio; los tests usan un `level.dat`
  sintético.
