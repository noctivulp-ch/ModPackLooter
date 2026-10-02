# 06 · Alcance y hoja de ruta

## MVP (fase 1) — implementado, pendiente de validar con DeceasedCraft

Objetivo: un sitio útil para un modpack real de una sola versión.

- Validado con **DeseacedCraft BetaCerrada** (Forge 1.20.1).
- Entrada: carpeta de instancia (`mods/`, `datapacks/`) + jar de vanilla
  (`--minecraft-jar` o `--download-vanilla`).
- **Minecraft 1.20.1** con **Forge** (y NeoForge 1.20.1, casi idéntico).
- Tablas de loot de **cofres de estructuras** (incl. tablas anidadas y tags).
- Estructuras → biomas (con tags resueltos).
- Asociación estructura↔loot por NBT y processor lists (exacta), tabla de
  conocimiento vanilla (conocida) y por nombre (heurística).
- Todas las loot tables listadas, aunque no tengan origen conocido (descubridor genérico).
- Puerta de descubrimiento con piezas enganchables ([ADR-0010](decisiones/0010-descubrimiento-por-ganchos-encadenados.md)).
- **Lootr 1.20.1**: contenedores por jugador, refresh y decay ([11-lootr.md](11-lootr.md)).
- Probabilidad por tirada y por cofre, cantidades.
- Nombres desde archivos `lang` (es/en).
- Sitio estático: inicio con búsqueda, fichas de objeto, estructura y bioma.
- Comandos `build` y `scan`.
- Sitio compatible con GitHub/GitLab Pages; binarios en GitHub Releases.
- Página de diagnósticos.

## Fase 2

- Iconos de objetos desde texturas.
- Entradas `.zip` de CurseForge y `.mrpack` de Modrinth.
- Otras fuentes: mobs (con spawns por bioma), pesca, arqueología.
- **NeoForge 1.21.1** (carpetas en singular, componentes de item).
- Global Loot Modifiers (Forge/NeoForge).
- Overrides manuales vía archivo de configuración.
- **Mods y configs que desactivan estructuras o spawns** (datapacks que vacían
  `structure_set`, Structurify, InControl, KubeJS…), de forma genérica, no por mod.
  Referencia: el port público del usuario a 1.21.1
  (`gitlab.com/nf1190525/deceasedcraft-revulpe-dev`).
- Interfaz del sitio traducible (hoy solo los nombres siguen el idioma; la interfaz está
  en español).
- Comandos de consulta (`where`, `structure`, `biome`) y `export`.

## Fase 3

- **TUI** interactiva.
- KubeJS / CraftTweaker (casos comunes).
- **26.1 → 26.3** (Forge y NeoForge).
- **Fabric**.
- `init --ci github|gitlab`.
- **Mod volcador** opcional ([ADR-0004](decisiones/0004-app-externa-con-volcador-opcional.md)).
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
