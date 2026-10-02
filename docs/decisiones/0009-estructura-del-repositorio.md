# ADR-0009: Repositorio con carpetas separadas para la app y el mod

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

La app externa (Go) y el futuro mod volcador (Java, [ADR-0004](0004-app-externa-con-volcador-opcional.md))
tienen ciclos de compilación, herramientas y releases distintos, pero
comparten la documentación y el contrato de datos.

## Decisión

Monorepo con esta estructura:

```
/docs/     documentación y ADR (común)
/app/      programa en Go (módulo propio, go.mod)
/mod/      futuro mod volcador (Gradle, MultiLoader-Template + Stonecutter)
```

- Cada carpeta tiene su propio README, build y CI (filtrado por rutas).
- Las etiquetas de release llevan prefijo: `app/vX.Y.Z` y `mod/vX.Y.Z`.
- El contrato entre ambas (formato del volcado) se documenta en `docs/` y
  se versiona aparte.

## Consecuencias

- Un cambio en el formato del volcado puede tocar ambas carpetas en el mismo PR.
- La CI de cada parte solo se ejecuta cuando cambian sus archivos.
