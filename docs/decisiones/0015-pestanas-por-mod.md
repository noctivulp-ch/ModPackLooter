# ADR-0015: Pestañas dedicadas por mod, solo si el mod está

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Lost Cities, la pesca (Starcatcher, Tide) o los tradeos no se entienden bien
como «estructura → tabla de loot»: tienen capas propias (estilos, edificios,
cebos, condiciones). El usuario pidió una pestaña por cada uno y que **no
aparezcan de ninguna forma** cuando el mod no está.

## Decisión

Cada descubridor puede adjuntar un modelo propio con `Claims.Attach(id, v)`;
el análisis lo expone en `Result.Extras` y el sitio genera una `Section` (entrada
de navegación + páginas + entradas de búsqueda) solo si encuentra ese modelo
con contenido. Los nombres se buscan en todos los `lang` posibles antes de
mostrar un ID ([doc 15](../15-pestanas-por-mod-y-lost-cities.md)).

## Alternativas consideradas

- **Que el análisis llame a cada mod directamente** — más simple, pero acopla
  el núcleo a mods concretos.
- **Pestañas fijas con un mensaje «mod no presente»** — descartado por el
  usuario: ruido para quien no tiene el mod.
- **Mostrar solo IDs técnicos** — descartado: el lector del sitio no es técnico.

## Consecuencias

- Añadir una pestaña = descubridor que adjunta + vista en `site`; nada más cambia.
- `site` conoce los modelos de los mods con pestaña (dependencia aceptada: es
  la capa de presentación de esos modelos).
- La búsqueda por clave de idioma «parecida» puede tomar un texto no pensado
  como nombre (p. ej. el título de un logro); se prefieren las claves con
  `title`/`name` y se descartan descripciones.
