# 17 · Cambios de mods a las fuentes

Los modpacks cambian el loot, los drops, los tradeos y la pesca de muchas
formas. La app no puede tener una regla para cada mod, así que hay una **capa
genérica** que detecta cambios en cualquier modpack, y **detectores
específicos** que leen formatos conocidos y dan mejor detalle. Los específicos
corren primero; los genéricos atrapan el resto y saltan lo que un específico ya
explicó (mismo archivo y línea, `Changes.Covered`).

Es una cuarta puerta de enganche (`ChangeDetector`), con el mismo registro,
fases y variantes que las demás. Cada cambio (`domain.Change`) dice:

- **qué toca**: una tabla, un tipo de tablas (cofres, bloques…) o todo el loot,
  los drops de una criatura (o de todas), los tradeos de una profesión (o de
  todas), la pesca o el trueque con piglins;
- **qué hace**: añade, quita, reemplaza, modifica, desactiva o «puede cambiar»;
- **objetos** afectados, tabla inyectada y probabilidad si se conocen;
- **seguro** (leído de datos o de un script que la app entiende) o **posible**
  (un indicio cuyo efecto exacto no se sabe);
- **quién** (mod, datapack, KubeJS…) y **de dónde** (archivo y línea).

## Formas cubiertas

| Forma de modificar | Detector | Nivel |
|---|---|---|
| Datapack o mod que reemplaza una tabla (mismo id en un pack posterior) | `table-overrides`: compara la tabla original con la efectiva: objetos que quita o añade y funciones que cambian (nivel de encantamiento, tesoro, cantidades) | seguro |
| Global Loot Modifiers de Forge/NeoForge (cualquier mod) | `global-loot-modifiers`: lista activa (`global_loot_modifiers.json` de todos los packs), tablas objetivo (`loot_table_id`, también dentro de `any_of`), probabilidad (`chance`, `random_chance`; 0 = desactivado), objetos y tablas que menciona; efecto por el nombre del tipo | seguro (posible si no se reconoce el efecto o no hay tabla objetivo) |
| LootJS (KubeJS) 2 y 3 | `lootjs`: tablas, tipos, bloques o criaturas objetivo; `addLoot`, `removeLoot`, `replaceLoot`, `modifyLoot`, `triggerLootTable`…; condiciones | seguro |
| Eventos de loot de KubeJS (`ServerEvents.*LootTables`) | `kubejs-loot-events`: `addX` redefine la tabla, `modify` la cambia | seguro |
| Tradeos en scripts (MoreJS y similares) | `trade-scripts`: `addTrade`, `removeTrades`, `removeVanillaTrades`… y profesión | seguro |
| CraftTweaker (`scripts/*.zs`) | `crafttweaker`: modificadores de loot y `villagerTrades` / `wanderingTrades` | seguro |
| Cualquier otra línea de script sobre loot, drops, tradeos, pesca o trueque | `script-mentions` (genérico) | posible |
| Opciones de config de cualquier mod (`config/`, `defaultconfigs/`, `serverconfig/`) | `config-keys` (genérico): claves cuyo nombre habla de loot, drops, tradeos, pesca o trueque y cambian algo; usa el comentario de la clave como explicación | posible |
| Código Java que cambia fuentes (cualquier mod) | `code-hooks` (genérico): busca en las clases de cada jar `LootTableLoadEvent`, `LivingDropsEvent`, `VillagerTradesEvent`, `WandererTradesEvent`, `ItemFishedEvent`, sus equivalentes de NeoForge/Fabric y mixins sobre `LootTable`, `VillagerTrades`, `FishingHook` y `PiglinAi` | posible |

Con esto se cubren las formas más comunes de modificar fuentes en Forge,
NeoForge y Fabric: datapacks, modificadores de loot, KubeJS/LootJS/MoreJS,
CraftTweaker, configs y código. Lo que queda fuera (por ejemplo, un mod que
cambia el loot sin usar ninguno de esos eventos ni mixins) es raro, y al menos
su config suele aparecer en `config-keys`.

### Filtros del genérico de configs

- Solo claves cuyo nombre (separado en palabras: `dropChance` → drop, chance)
  nombra una fuente, y que además cambian algo (disable, chance, weight,
  multiplier…) o tienen una lista.
- Se descartan claves que hablan de otra cosa (spawn, sound, equip, fluid,
  render…), claves que son ids (`"minecraft:villager": …`), configs de cliente
  y las de mods con detector propio (Lootr, Lost Cities, Tide, Starcatcher…).
- `disable… = false` y `enable… = true` se ignoran (no cambian nada).
- Si la clave no nombra la fuente pero desactiva algo, se mira su comentario
  («Disable Iron Farms» → «Makes Iron Golems not drop Iron Ingots»).

## En el sitio

- Página **Cambios de mods** (`cambios/`), en la navegación solo si hay cambios:
  agrupados por tipo de fuente, seguros primero; los cambios que vienen del
  mismo origen y tocan muchas tablas se muestran en una fila desplegable.
- Las fichas de **tabla** muestran sus cambios (también los de su tipo, p. ej.
  «todos los cofres»), y las de **objeto** los cambios que lo añaden o lo quitan.
- Los modificadores desactivados (probabilidad 0) solo aparecen en la página
  de cambios, atenuados.

## Específicos que mejoran el formato

La lista crece según haga falta: cuando un mod tiene un formato propio que el
genérico solo ve como «posible», se añade un detector específico que lo lee y
marca su origen como cubierto. Ejemplos previstos: opciones de Quark
(`game_nerfs`), tradeos por config de mods como Vinery.

## Caso real (DeceasedCraft)

- DCTweaks reemplaza el tesoro de pesca (sin encantamientos de tesoro: no da
  Reparación) y las tablas de zombi, husk y drowned (quita hierro, zanahoria,
  papa; añade dinero, hueso, hilo).
- Modificadores: Tide añade cañas a cofres vanilla; Starcatcher, sombreros a
  naufragios; DCTweaks desactiva los de fancytrinkets (probabilidad 0).
- LootJS: cantidades de los cofres según la dificultad.
- Configs: Quark sin lana de ovejas ni hierro de gólems, Sophisticated
  Backpacks sin loot en cofres, Vinery con tradeos propios…
