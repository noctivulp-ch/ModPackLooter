# 18 · Pestaña de tradeos

Capas: **Tradeos › Sistema › Comerciante › Nivel › Oferta**.

## Aldeanos y comerciante errante (vanilla)

- Están en el código del juego, no en datos: la app los trae como conocimiento
  incluido (confianza «Conocida»), **verificados contra los jars de cliente
  1.20.1 y 1.21** con un intérprete de bytecode (ver
  [`tools/villager-trades`](../tools/villager-trades/README.md)).
- 13 profesiones × 5 niveles, con precios, usos y experiencia exactos. Cada
  nivel aprende **2 ofertas al azar** de su lista: la probabilidad de ofrecer
  cada una es 2 / n (100 % si hay 2 o menos).
- El granjero experto tiene 7 ofertas (tarta y 6 estofados sospechosos).
- Comerciante errante: 5 de 64 ofertas comunes y 1 de 6 raras, con precios exactos.
- Variantes por versión: `Trades1_20` (1.20.x) y `Trades1_21` (1.21.x: mapa
  de cámaras de prueba del cartógrafo, `turtle_scute`).

## Trueque con piglins

Es la tabla `minecraft:gameplay/piglin_bartering`: sale de los datos del
modpack (con sus reemplazos y modificadores), así que se enlaza su página.

## NPCs (CustomNPCs)

- Lee los NPCs guardados (`customnpcs/clones/**.json` del modpack y del
  mundo modelo). Están en SNBT con forma de JSON (`5b`, `1.0f`, saltos de
  línea dentro de textos): se limpian antes de leerlos.
- Drops: `NpcInv` con la probabilidad de `DropChance` por ranura.
- Comerciantes (`Role` = 1, `RoleType.TRADER` en la
  [API de scripting](https://goodbird-git.github.io/CNPC-Unofficial-1.20.1-ScriptingDoc/)):
  por ranura, dos monedas y un objeto vendido (`IRoleTrader.getCurrency1/2`,
  `getSold`), guardados como `TraderCurrency` (ranuras i e i+18) y
  `TraderSold`. Si usan un mercado compartido (`getMarket`, `TraderMarket`), se
  lee de `customnpcs/markets/<nombre>.json` del mundo indicado con `--world`;
  sin mundo se avisa en la página del NPC.
- Tiendas de otros mods guardadas en el NPC: cualquier entrada de `ForgeData`
  cuyo nombre termina en `shop.json` y trae una lista de productos (p. ej.
  dochi_rpg_maker, `type: npc_shop`): objeto, cantidad, precio y moneda del
  producto (o la de la tienda si es `inherit`), variante (nombre) y existencias.
  En DeceasedCraft son 11 NPCs con 179 productos.
- Los scripts de CustomNPCs (`customnpcs/scripts`) entran en el genérico de
  scripts de la capa de cambios (pueden reaccionar a `RoleEvent.TraderEvent`).
- El nombre del NPC puede ser una clave de idioma (`npc.mod.bandit.name`).

## Cambios de mods

Los cambios a tradeos y trueque que detecta la capa genérica
([doc 17](17-cambios-de-fuentes.md)) aparecen en la portada de tradeos y en la
página de cada profesión afectada.

## En la ficha de objeto

- **Por tradeo**: quién lo vende (o lo compra), a qué precio y con qué
  probabilidad de ofrecerlo.
- **Lo sueltan NPCs**: NPC, cantidad y probabilidad.
