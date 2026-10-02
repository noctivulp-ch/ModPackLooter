# ModPackLooter · app

Programa de línea de comandos en Go. Analiza un modpack y genera el sitio
estático. Arquitectura y decisiones en [`../docs`](../docs/README.md).

## Requisitos

- Go 1.24 o superior (solo para compilar; el binario no tiene dependencias).

## Compilar y probar

```bash
cd app
go build -o modpacklooter ./cmd/modpacklooter
go test ./...
```

## Uso (estado actual)

```bash
./modpacklooter --version
./modpacklooter plan --mc-version 1.20.1 --loader forge --mod lootr
./modpacklooter plan --json
```

`plan` muestra qué descubridores de la puerta de enganche se ejecutarían y en
qué orden. Los comandos `scan` y `build` llegarán con la primera versión.

## Estructura

```
cmd/modpacklooter/      raíz de composición: registra adaptadores y descubridores
internal/domain/        conceptos del dominio (sin dependencias externas)
internal/mcversion/     versiones de Minecraft (1.20.1 … 26.3) y rangos
internal/discovery/     puerta de descubrimiento: registro, plan, reclamaciones
internal/discovery/*/   un paquete por descubridor (generic, …)
internal/cli/           adaptador de línea de comandos (Cobra)
```
