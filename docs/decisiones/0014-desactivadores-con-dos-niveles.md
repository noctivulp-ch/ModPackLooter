# ADR-0014: Desactivadores con dos niveles de confianza

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Los modpacks desactivan estructuras, biomas y spawns con datapacks, configs de mods
(Structurify, Biome Replacer, InControl…) y scripts (KubeJS). Si el sitio muestra loot de
estructuras que no existen, engaña al jugador. No todas las fuentes se pueden interpretar
con certeza.

## Decisión

- Una tercera puerta de enganche, **desactivadores** (`Disabler`), con el mismo registro,
  fases y variantes que el descubrimiento.
- Dos niveles: **desactivado** (seguro: datos o config entendida por completo) y
  **posiblemente desactivado** (indicios).
- Además de los específicos por mod, dos **genéricos** (`config-mentions`,
  `kubejs-scripts`) detectan menciones junto a palabras de desactivación sin conocer el mod;
  por eso solo dan "posible".
- Lo desactivado con seguridad no cuenta para "mejores fuentes"; lo posible solo se avisa.

Detalle: [14-mundo-modelo-y-desactivadores.md](../14-mundo-modelo-y-desactivadores.md).

## Consecuencias

- Añadir soporte para otro mod que desactiva cosas = añadir un plugin.
- Los genéricos pueden dar falsos positivos; por eso nunca dan "seguro" y explican la línea
  y la clave que los dispararon.
