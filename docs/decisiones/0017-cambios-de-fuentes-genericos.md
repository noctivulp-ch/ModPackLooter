# ADR-0017: Cambios a las fuentes: capa genérica + específicos

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Los mods cambian el loot, los drops, los tradeos y la pesca con datapacks,
modificadores de loot, scripts, configs y código. Soportar mod por mod no
escala: el usuario pidió algo procedural que cubra al menos el 70 % de las
formas comunes en cualquier modpack, con mods específicos solo para mejorar el
formato.

## Decisión

Una puerta de enganche `ChangeDetector`. Detectores de formato conocido
(reemplazo de tablas, Global Loot Modifiers, LootJS, eventos de loot de KubeJS,
tradeos en scripts, CraftTweaker) dan cambios **seguros**; los genéricos
(líneas de scripts, claves de config, eventos y mixins en el código de los
jars) dan cambios **posibles** y saltan lo que un específico ya explicó.

Detalle en [17-cambios-de-fuentes.md](../17-cambios-de-fuentes.md).

## Alternativas consideradas

- **Un plugin por mod** — no escala y deja fuera lo que no se conoce.
- **Solo indicios genéricos** — se perdería el detalle que sí se puede leer
  (qué objeto, qué tabla, qué probabilidad).

## Consecuencias

- Cualquier modpack obtiene una lista de cambios sin soporte específico.
- Los genéricos tienen falsos positivos: siempre son «posible» y muestran la
  clave, el comentario o la línea que los disparó.
- Revisar el código de los jars añade unos segundos al análisis.
