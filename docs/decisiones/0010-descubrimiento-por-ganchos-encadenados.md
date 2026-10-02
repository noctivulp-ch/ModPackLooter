# ADR-0010: Descubrimiento mediante piezas enganchables encadenadas

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Hay que soportar muchas fuentes de loot (plantillas NBT, processor lists,
entidades, mobs, pesca, Lootr…), cada una con particularidades por versión
de Minecraft y por cargador. Hace falta poder añadirlas sin tocar el núcleo
y sin duplicar resultados.

## Decisión

- La etapa de descubrimiento es una **puerta de enganche** con un registro
  de `Discoverer`, uno por fuente.
- Las piezas se ejecutan en orden por **fase** (específicos → relacionales →
  heurísticos → genérico) y por **prioridad**, con desempate por ID.
- Cada pieza declara su **aplicabilidad**: rango de versiones, cargadores y
  mods requeridos. Las variantes por versión son tipos distintos con el
  mismo ID.
- Los resultados se publican como **reclamaciones** con nivel de confianza;
  se deduplican conservando la más fiable.
- Un **descubridor genérico**, siempre el último, solo procesa las loot
  tables sin reclamar.
- Las piezas que solo anotan (Lootr, probabilidades) son **enriquecedores**,
  en una segunda puerta con el mismo mecanismo.
- El registro es **explícito** en la raíz de composición, sin `init()`.

Detalle e investigación: [12-arquitectura-de-descubrimiento.md](../12-arquitectura-de-descubrimiento.md).

## Alternativas consideradas

- **Un único analizador grande** con condicionales por versión: rápido al
  inicio e inmantenible después.
- **Registro global con `init()`** (estilo `database/sql`): menos código de
  arranque, pero estado global y pruebas más difíciles.
- **Eventos sin orden** (*pub/sub* puro): no garantiza que el genérico vaya al
  final ni que el resultado sea determinista.

## Consecuencias

- Añadir una fuente o una versión = añadir una pieza y registrarla.
- Hace falta un test que valide el plan para cada versión soportada.
- `scan --explain` mostrará el plan de ejecución para depurar.
