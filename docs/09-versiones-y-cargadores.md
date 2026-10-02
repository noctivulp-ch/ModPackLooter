# 09 · Versiones de Minecraft y cargadores

Meta: **1.20.1 → 26.3**. Prioridad de cargadores: **Forge y NeoForge**;
**Fabric** en el futuro. Decisión: [ADR-0006](decisiones/0006-rango-de-versiones-y-cargadores.md).

> Las filas marcadas con *(verificar)* se confirmarán con fixtures reales antes
> de implementar el adaptador correspondiente.

## Diferencias de formato que importan a la app

| Versión | Cambios relevantes para leer loot/estructuras |
|---|---|
| **1.20.1** (*pack_format* 15) | Carpetas en plural: `loot_tables/`, `structures/`, `tags/items/`. Los items llevan NBT (`set_nbt`). Existe `append_loot` en processor lists. **Versión del MVP.** |
| 1.20.2 – 1.20.4 | NeoForge se separa de Forge (namespace `neoforge` en sus datos). *(verificar)* |
| 1.20.5 – 1.20.6 | **Componentes de item** sustituyen al NBT (`set_components`, `set_custom_data`). |
| **1.21** | Carpetas en **singular**: `loot_table/`, `structure/`, `tags/item/`, `tags/block/`. Trial Chambers y bóvedas (`vault`). |
| 1.21.x | Ajustes menores en funciones/condiciones de loot. |
| **26.1** | Nueva numeración, juego sin ofuscar (no afecta a la app) y tipo de loot table `villager_trade`. *pack_format* 101.1. |
| 26.2 | Por revisar con el changelog. |
| **26.3** | `rolls`/`bonus_rolls` aceptan un ID de *number provider*; registros con referencias a elementos y tags; `set_loot_table`: `name` → `loot_table_id`. *pack_format* 121.0. |

**Estrategia:** un único modelo de dominio y **adaptadores de formato por
rango de versiones**. Cada adaptador normaliza lo que lee al modelo común.
La versión se detecta por los metadatos del modpack o del cargador, y se puede
forzar con `--mc-version`.

## Diferencias entre cargadores

| Aspecto | Forge | NeoForge | Fabric (futuro) |
|---|---|---|---|
| Metadatos del mod | `META-INF/mods.toml` | `META-INF/mods.toml` (≤1.20.4) → `META-INF/neoforge.mods.toml` *(verificar desde qué versión)* | `fabric.mod.json` |
| Global Loot Modifiers | `data/forge/loot_modifiers/global_loot_modifiers.json` + `data/<ns>/loot_modifiers/*.json` | Igual con namespace `neoforge` (en 1.20.1, `forge`) *(verificar)* | No existe; los mods usan eventos en código → no visibles. |
| Biome modifiers (spawns) | `data/<ns>/forge/biome_modifier/` | `data/<ns>/neoforge/biome_modifier/` *(verificar)* | API de Fabric en código. |
| Jar-in-jar | `META-INF/jarjar/` | `META-INF/jarjar/` | `META-INF/jars/` |

> Los mods incluidos dentro de otros (jar-in-jar) también pueden aportar
> datos, así que el lector de recursos los recorre de forma recursiva.

## Datos de vanilla

El loot vanilla no está en la carpeta del modpack. Opciones, en orden:

1. `--minecraft-jar <ruta>` al jar del cliente o del servidor que ya tiene el
   launcher (por ejemplo, `versions/1.20.1/1.20.1.jar`).
2. `--download-vanilla`: se descarga del manifiesto oficial de Mojang y se
   guarda en caché local. Nunca se redistribuye.
3. Sin vanilla: el sitio solo muestra contenido de mods y avisa en los
   diagnósticos.

## Estructuras vanilla generadas por código

Mineshaft, fortaleza del Nether, stronghold, templos, monumento, tesoro
enterrado, naufragios (*data markers*), etc. no declaran su loot en datos.
La app incluye una **tabla de conocimiento por versión** (estructura → loot
tables), mantenida a mano y versionada en el repo, con confianza `conocida`.
