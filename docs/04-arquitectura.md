# 04 · Arquitectura

> Este documento es independiente del lenguaje. Las tecnologías concretas se
> decidirán en [07-tecnologias.md](07-tecnologias.md).

## Estilo: hexagonal (puertos y adaptadores)

El **núcleo** (dominio + casos de uso) no depende de nada externo. Todo lo que
toca el mundo exterior (sistema de archivos, `.jar`, NBT, HTML, terminal) es
un **adaptador** que implementa un **puerto** (interfaz) definido por el
núcleo.

```
            ┌──────────────── Interfaces (pieles) ────────────────┐
            │        CLI            TUI (futuro)      ¿API/GUI?   │
            └───────────────┬────────────────────────────────────┘
                            │ llama a
            ┌───────────────▼──────── Aplicación ─────────────────┐
            │  Casos de uso: BuildSite, ScanModpack, InspectItem   │
            │  Puertos (interfaces) que necesitan los casos de uso │
            └───────────────┬────────────────────────────────────┘
                            │ usa
            ┌───────────────▼────────── Dominio ──────────────────┐
            │  Entidades, objetos de valor, resolución del grafo, │
            │  cálculo de probabilidades, reglas de prioridad     │
            └─────────────────────────────────────────────────────┘
                            ▲ implementan los puertos
            ┌───────────────┴─────── Infraestructura ─────────────┐
            │ Lectores (carpeta, zip, jar, mrpack, CurseForge)    │
            │ Parsers por versión (JSON loot, NBT, tags, lang)    │
            │ Renderizadores (sitio HTML, export JSON)            │
            │ Reportadores de progreso, logging, caché            │
            └─────────────────────────────────────────────────────┘
```

**Regla de dependencias:** las flechas apuntan siempre hacia el dominio.
`Dominio` no importa nada de las demás capas; `Aplicación` solo importa
`Dominio`; `Infraestructura` e `Interfaces` dependen de las capas internas,
nunca al revés.

## Pipeline

```
 Fuente del modpack
        │  ModpackSource
        ▼
 1. Descubrimiento   → lista de “paquetes de recursos” (vanilla, mods, datapacks, scripts)
        │  ResourceProvider
        ▼
 2. Extracción       → archivos virtuales con prioridad (el último gana)
        │  *Parser (por tipo y versión)
        ▼
 3. Parseo           → objetos de dominio sin resolver (referencias por ID)
        │  dominio
        ▼
 4. Resolución       → tags expandidos, tablas anidadas, estructuras↔piezas↔loot,
        │               modificadores, heurísticas, overrides
        ▼
 5. Indexado         → LootIndex (consultas en ambas direcciones) + Diagnostics
        │  SiteRenderer / Exporter
        ▼
 6. Salida           → sitio estático, JSON, informe
```

Cada etapa recibe datos inmutables y devuelve datos nuevos: así se pueden
probar por separado y, más adelante, cachear.

## Puertos principales (borrador)

| Puerto | Responsabilidad | Adaptadores previstos |
|---|---|---|
| `ModpackSource` | Descubrir los paquetes de recursos de un modpack. | Carpeta de instancia, `.zip` CurseForge, `.mrpack`, servidor. |
| `ResourceProvider` | Listar/leer archivos de un paquete. | Carpeta, zip/jar. |
| `LootTableParser` | Convertir JSON en `LootTable`. | Por rango de versiones de MC. |
| `StructureParser` | Leer definiciones de estructura y pools. | Por rango de versiones. |
| `StructureTemplateReader` | Extraer contenedores con `LootTable` de `.nbt`. | Lector NBT. |
| `TagRepository` | Resolver tags de biomas y objetos. | Desde los datos fusionados. |
| `LootModifierSource` | Aportar modificaciones extra al loot. | GLM, KubeJS, CraftTweaker (fases futuras). |
| `LocalizationProvider` | Traducir IDs a nombres. | Archivos `lang`. |
| `IconProvider` | Obtener/generar el icono de un objeto. | Texturas de `assets/`. |
| `AssociationStrategy` | Relacionar loot ↔ estructura cuando no hay dato exacto. | NBT exacto, heurística por nombre, overrides manuales. |
| `SiteRenderer` | Generar la salida a partir del `LootIndex`. | Sitio HTML, export JSON. |
| `ProgressReporter` | Informar avance y diagnósticos. | Barra de CLI, TUI, silencioso (tests). |
| `ConfigLoader` | Cargar la configuración del proyecto. | Archivo de config + flags. |

## SOLID aplicado

| Principio | Cómo se aplica aquí |
|---|---|
| **S** — Responsabilidad única | Cada etapa del pipeline es una pieza. Leer un `.jar`, parsear una tabla, calcular probabilidades y pintar HTML son clases/módulos distintos. |
| **O** — Abierto/cerrado | Soportar NeoForge, MC 1.22 o un export a Markdown = **nuevo adaptador** registrado, sin editar el núcleo. Las estrategias de asociación se encadenan sin modificar las existentes. |
| **L** — Sustitución de Liskov | Cualquier `ModpackSource` (carpeta, zip, mrpack) es intercambiable; los casos de uso no saben cuál reciben. Lo mismo para `ProgressReporter` (CLI vs TUI). |
| **I** — Segregación de interfaces | Puertos pequeños y específicos (`LocalizationProvider` no tiene nada que ver con `IconProvider`), en lugar de un gran `ModpackReader`. |
| **D** — Inversión de dependencias | Los casos de uso dependen de interfaces; la composición concreta se hace en un único punto de arranque (*composition root*) en cada interfaz (CLI/TUI). |

## Por qué esto permite la TUI futura

La CLI y la TUI solo:

1. traducen la entrada del usuario a una petición de caso de uso
   (`BuildSiteRequest`, `InspectItemRequest`…),
2. aportan su propio `ProgressReporter`,
3. presentan el resultado.

Ninguna lógica de negocio vive en la interfaz. Ver
[ADR-0002](decisiones/0002-cli-primero-nucleo-independiente.md).

## Errores y diagnósticos

- Un archivo malformado **no detiene** el análisis: se registra un
  `Diagnostic` (nivel, archivo, mod, mensaje) y se continúa.
- Errores fatales (ruta inexistente, versión no soportada) sí detienen el
  proceso con un mensaje claro y un código de salida distinto de cero.
- Los diagnósticos se muestran en la CLI y en la página `/acerca/` del sitio.

## Pruebas

- **Dominio y resolución:** pruebas unitarias con datos en memoria.
- **Parsers:** pruebas con *fixtures* reales pequeños (JSON/NBT) por versión.
- **Pipeline completo:** un mini-modpack de prueba en el repositorio y
  comparación del sitio/JSON generado (*snapshot*).
- **Contratos de puertos:** una misma batería de pruebas que deben superar
  todos los adaptadores de un puerto (refuerza Liskov).
