# ADR-0003: La salida principal es un sitio web estático

- **Estado:** Aceptada
- **Fecha:** 2026-10-01

## Contexto

Los jugadores necesitan consultar el loot de forma cómoda, y los creadores de
modpacks quieren publicarlo sin mantener infraestructura.

## Decisión

La app genera un sitio **100 % estático** (HTML, CSS, JS y un índice JSON de
búsqueda) que funciona sin servidor, incluso abierto desde `file://`.

## Alternativas consideradas

- **Aplicación web con backend y base de datos** — consultas más potentes,
  pero exige hosting y mantenimiento.
- **Solo exportar JSON** — útil para herramientas, inútil para jugadores.

## Consecuencias

- Publicable en GitHub Pages, Netlify o compartible como `.zip`.
- La búsqueda debe resolverse en el navegador con un índice precalculado;
  hay que vigilar su tamaño en modpacks grandes.
- El renderizador del sitio es un adaptador más: se podrán añadir otras
  salidas (JSON, Markdown) sin tocar el núcleo.
