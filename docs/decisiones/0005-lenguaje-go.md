# ADR-0005: Implementar la app en Go

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

Se necesita leer jars/zip, JSON y NBT; distribuir un binario autocontenido
multiplataforma; tener CLI ahora y TUI después; y facilitar una arquitectura
SOLID desacoplada. No hay preferencia previa de lenguaje. Comparativa en
[07-tecnologias.md](../07-tecnologias.md).

## Decisión

Usar **Go**, con Cobra para la CLI, Bubble Tea para la futura TUI,
`html/template` + `embed` para el sitio y GoReleaser para publicar.

## Alternativas consideradas

- **Kotlin/Java**: el ecosistema de Minecraft y la opción de compartir código
  con un mod; pero el binario nativo (GraalVM) es más costoso y la TUI es más
  débil.
- **Rust**: excelente rendimiento y TUI (Ratatui), con más curva de
  aprendizaje y fricción para iterar rápido.
- **Python**: rápido de escribir, pero la distribución autocontenida es frágil.

## Consecuencias

- `go build` produce un binario sin dependencias; *cross-compile* sencillo.
- La comunicación con un futuro mod volcador será por contrato JSON, no por
  código compartido.
- NBT: se valida una librería existente o se escribe un lector mínimo propio.
