# ADR-0008: Ciclo de iteración con un modpack real

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

La calidad del resultado solo se puede juzgar con un modpack real, que no
puede subirse al repositorio.

## Decisión

- Modpack de referencia: **DeseacedCraft BetaCerrada** (Forge 1.20.1).
- El usuario compila (o descarga de Releases), ejecuta la app sobre el
  modpack, revisa el sitio y devuelve un **zip del sitio + un mensaje de
  review**.
- Cada review se convierte en tareas y, si implica decisiones, en ADR.
- El sitio generado incluye `diagnostics.json` y la versión de la app, para
  que cada zip sea autoexplicativo.

Proceso detallado en [10-proceso-de-pruebas-y-entrega.md](../10-proceso-de-pruebas-y-entrega.md).

## Consecuencias

- La app debe producir diagnósticos ricos, porque no hay acceso directo al
  modpack.
- Los fixtures automáticos cubren la regresión; el modpack real valida la
  utilidad.
