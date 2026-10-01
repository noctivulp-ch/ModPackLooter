# 07 · Tecnologías — PENDIENTE DE DISCUTIR

> Estado: **abierto**. Este documento recoge los criterios y las preguntas
> que hay que responder. Cuando se decida, cada elección quedará como un ADR
> en [`decisiones/`](decisiones/).

## Qué hay que decidir

1. **Lenguaje del núcleo y la CLI.**
2. **Librería de CLI** (parseo de argumentos, ayuda, autocompletado).
3. **Librería de TUI** futura (conviene que encaje con el lenguaje elegido).
4. **Lectura de NBT** (formato binario comprimido de las plantillas).
5. **Lectura de zip/jar** en streaming.
6. **Generación del sitio**: plantillas en el servidor de build vs. sitio
   cliente que consume un JSON.
7. **Frontend del sitio**: HTML/CSS puro, framework ligero, CSS utilitario…
8. **Búsqueda en el cliente** (índice precalculado, búsqueda difusa).
9. **Formato del archivo de configuración** (TOML, YAML, JSON).
10. **Distribución**: binario único, paquete de un gestor, contenedor.
11. **Herramientas de calidad**: tests, linter, formateo, CI.
12. **Versión de Minecraft objetivo del MVP.**

## Criterios de evaluación

| Criterio | Por qué importa |
|---|---|
| Ecosistema para NBT/zip/JSON | Es el corazón del análisis. |
| Calidad de librerías CLI **y** TUI | Requisito explícito: CLI ahora, TUI después. |
| Facilidad para aplicar SOLID/hexagonal | Interfaces, inyección de dependencias, tipado. |
| Distribución a usuarios no técnicos | Creadores de modpacks en Windows/macOS/Linux. |
| Rendimiento con modpacks grandes | Cientos de jars. |
| Familiaridad del equipo | Velocidad de desarrollo y mantenimiento. |

## Preguntas para la próxima conversación

- ¿Qué lenguajes conoces o prefieres?
- ¿Es importante distribuir un **ejecutable único** sin dependencias?
- ¿Qué modpack/versión de Minecraft usaremos como caso de prueba real?
- ¿Loader principal: Forge, NeoForge, Fabric o varios?
- ¿El sitio debe poder publicarse en GitHub Pages desde el propio repo del
  modpack (p. ej. vía GitHub Action)?
