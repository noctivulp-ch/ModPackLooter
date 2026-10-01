# 06 · Alcance y hoja de ruta

## MVP (fase 1)

Objetivo: un sitio útil para un modpack real de una sola versión.

- Entrada: carpeta de instancia (`mods/`, `datapacks/`) + jar de vanilla
  opcional.
- Una versión de Minecraft objetivo (a decidir; ver
  [07-tecnologias.md](07-tecnologias.md)).
- Tablas de loot de **cofres de estructuras** (incl. tablas anidadas y tags).
- Estructuras → biomas (con tags resueltos).
- Asociación estructura↔loot por NBT (exacta) y por nombre (heurística).
- Probabilidad por tirada y por cofre, cantidades.
- Nombres desde archivos `lang` (es/en).
- Sitio estático: inicio con búsqueda, fichas de objeto, estructura y bioma.
- Comandos `build` y `scan`.
- Página de diagnósticos.

## Fase 2

- Iconos de objetos desde texturas.
- Entradas `.zip` de CurseForge y `.mrpack` de Modrinth.
- Otras fuentes: mobs, pesca, arqueología, bóvedas.
- Global Loot Modifiers (Forge/NeoForge).
- Overrides manuales vía archivo de configuración.
- Comandos de consulta (`where`, `structure`, `biome`) y `export`.

## Fase 3

- **TUI** interactiva.
- KubeJS / CraftTweaker (casos comunes).
- Soporte multi-versión de Minecraft.
- Personalización visual del sitio, temas.
- Caché incremental para regenerar rápido.
- Comparar dos versiones de un modpack (*diff* de loot).

## Riesgos conocidos

| Riesgo | Mitigación |
|---|---|
| Loot asignado por código, no visible en datos. | Heurísticas + overrides + marcar “no determinado”. |
| Formatos distintos entre versiones de MC. | Parsers aislados por versión detrás de un puerto. |
| Scripts (KubeJS/CraftTweaker) son código arbitrario. | Soportar solo patrones comunes; el resto, diagnóstico. |
| Modpacks grandes (300+ mods) = mucho I/O. | Lectura en streaming, paralelismo, caché. |
| Licencias de texturas/assets al publicar el sitio. | Iconos opcionales y aviso en la documentación. |
| Probabilidades con condiciones complejas. | Marcar como “aproximada” en lugar de omitir. |
