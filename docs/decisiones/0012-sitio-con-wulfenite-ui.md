# ADR-0012: El sitio usa Wulfenite UI, PC primero

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

El sitio necesita una identidad propia que el creador del modpack pueda personalizar. El
usuario tiene un sistema de diseño propio, **Wulfenite UI**, con modos oscuro y claro y tres
metales. Los jugadores consultan sobre todo desde el PC.

## Decisión

- La identidad visual del sitio sigue los tokens y reglas de Wulfenite UI.
- Personalización: título, logo y metal (`oro`, `jade`, `peltre`) por configuración o flags.
- Diseño **PC primero** con soporte aceptable en móvil.
- Idioma por defecto español; inglés opcional.

Detalle: [13-diseno-del-sitio.md](../13-diseno-del-sitio.md).

## Consecuencias

- Los valores de color se copian de Wulfenite UI; si el sistema cambia, se actualiza
  `app/internal/site/assets/style.css`.
- Sin sombras ni degradados decorativos; la profundidad la dan los bordes.
