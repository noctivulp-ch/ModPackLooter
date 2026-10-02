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
- Comerciantes: ranuras `TraderSold` con su precio en `TraderCurrency`
  (ranuras i e i+18). Los mercados creados dentro de un mundo solo se ven con
  `--world`.
- El nombre del NPC puede ser una clave de idioma (`npc.mod.bandit.name`).

## Cambios de mods

Los cambios a tradeos y trueque que detecta la capa genérica
([doc 17](17-cambios-de-fuentes.md)) aparecen en la portada de tradeos y en la
página de cada profesión afectada.

## En la ficha de objeto

- **Por tradeo**: quién lo vende (o lo compra), a qué precio y con qué
  probabilidad de ofrecerlo.
- **Lo sueltan NPCs**: NPC, cantidad y probabilidad.
