# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

- **Jugadores de un modpack** que buscan dónde conseguir un objeto. Consultan el sitio sobre
  todo en el **PC, en una segunda pantalla o con alt-tab** mientras juegan; el móvil debe
  funcionar de forma aceptable, pero no es el uso principal.
- **Creadores de modpacks y servidores** que generan el sitio con la CLI y lo publican en
  GitHub Pages o GitLab Pages junto a su pack.

## Product Purpose

ModPackLooter analiza los archivos de un modpack de Minecraft y genera un sitio estático que
responde rápido a: "¿dónde consigo este objeto?", "¿qué hay en esta estructura?" y "¿qué
puedo saquear en este bioma?". El éxito es encontrar un objeto en segundos y entender el
resultado sin conocimientos técnicos.

## Positioning

Funciona sobre la combinación exacta de mods, datapacks y configuración de cada modpack (no
sobre una wiki genérica) y se genera fuera del juego. Indica qué tan fiable es cada dato
(exacta, conocida, heurística o desconocida) en lugar de inventarlo.

## Operating Context

- Se genera con la CLI sobre la carpeta de la instancia; el resultado se abre desde `file://`
  o se publica en GitHub/GitLab Pages, siempre con enlaces relativos.
- Modpack de referencia: DeceasedCraft BetaCerrada (Forge 1.20.1, Lost Cities, Lootr).
- Ciclo de review: el usuario devuelve el sitio en un zip con comentarios.

## Capabilities and Constraints

- Sitio 100 % estático, sin backend y funcionando offline; búsqueda en el navegador con un
  índice precalculado.
- Idioma por defecto: español; inglés opcional (`--lang`).
- Iconos de objetos: pendientes (fase 2).

## Brand Commitments

- Identidad visual propia basada en el sistema de diseño **Wulfenite UI** del usuario
  (laca + metal, modos oscuro y claro, tres metales), con personalización por el creador del
  modpack: título, logo y metal.

## Evidence on Hand

- Datos vanilla reales (misode/mcmeta) para pruebas; ningún testimonio ni métrica de uso.

## Product Principles

1. Encontrar antes que explorar: la búsqueda siempre está a mano.
2. Probabilidades legibles: porcentajes y cantidades, nunca pesos crudos.
3. Honestidad con los datos: mostrar la confianza y lo no determinado.
4. Portable: un sitio que se abre igual en local que en cualquier hosting estático.
