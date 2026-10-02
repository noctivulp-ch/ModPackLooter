# ADR-0002: CLI primero, con un núcleo independiente de la interfaz

- **Estado:** Aceptada
- **Fecha:** 2026-10-01

## Contexto

La app debe usarse por línea de comandos desde el inicio y tener una TUI en el
futuro. Además, el código debe seguir SOLID y estar desacoplado.

## Decisión

Se adopta una arquitectura hexagonal (puertos y adaptadores):

- El **dominio** y los **casos de uso** no conocen ni la terminal, ni el
  sistema de archivos, ni el formato de salida.
- La **CLI** es un adaptador de entrada fino: parsea argumentos, construye la
  petición, inyecta dependencias (*composition root*) y presenta resultados.
- La futura **TUI** será otro adaptador de entrada que reutiliza los mismos
  casos de uso.
- El progreso se comunica mediante un puerto `ProgressReporter`, con una
  implementación por interfaz.

Detalle en [04-arquitectura.md](../04-arquitectura.md).

## Alternativas consideradas

- **Lógica dentro de los comandos de la CLI** — más rápido al principio, pero
  obliga a reescribir para la TUI y dificulta las pruebas.
- **Construir la TUI desde el inicio** — más trabajo antes de validar la idea.

## Consecuencias

- Algo más de estructura (interfaces, inyección) desde el primer día.
- El núcleo se puede probar sin terminal ni archivos reales.
- Añadir TUI, GUI o API más adelante no exige tocar la lógica de negocio.
