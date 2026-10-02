# 12 · Arquitectura de descubrimiento: la “puerta de enganche”

> Decisión: [ADR-0010](decisiones/0010-descubrimiento-por-ganchos-encadenados.md).

## Objetivo

La zona de descubrimiento debe funcionar como una **puerta de enganche**
(*hook*):

- Hay **una clase de código por cada fuente soportada** (cofres de
  plantillas NBT, processor lists, vagonetas, mobs, pesca, Lootr, etc.).
- Todas se **enganchan** a la puerta y se **ejecutan en orden**.
- Al final se ejecuta una **genérica** que recoge lo que nadie reclamó y
  **filtra las fuentes ya consideradas**.
- Cada pieza puede tener **variantes por versión** de Minecraft y por
  cargador, sin `if version …` repartidos por el código.

## 1. Arquitecturas similares investigadas

| Proyecto / patrón | Cómo lo resuelve | Qué tomamos |
|---|---|---|
| **Syft** (Anchore, Go) — genera SBOMs | Un *cataloger* por ecosistema (npm, rpm, jar…), cada uno declara qué archivos le interesan. Varios catalogers pueden encontrar el mismo paquete; una fase posterior crea la relación **ownership-by-file-overlap** y **elimina el duplicado menos fiable**. | Un descubridor por fuente; **deduplicación por reclamaciones**, conservando la evidencia más fiable. |
| **Trivy / fanal** (Aqua, Go) | Registro de **analyzers** con `Required(path)` (solo se abren los archivos que alguien pide) y **post-analyzers** que trabajan sobre lo ya analizado. Cada analyzer tiene `Type()` y `Version()`. | `Requires()` para no leer el jar entero; separar **descubrir** de **enriquecer**; versionar cada pieza. |
| **Caddy** (Go) | Módulos que se registran; los *handlers* forman una **cadena ordenada** con un orden por defecto y orden relativo `before`/`after`. | Orden declarativo por **fases** y prioridad dentro de cada fase. |
| **Chain of Responsibility** (GoF) | Cada eslabón decide si procesa la petición o la pasa al siguiente. | La genérica final procesa solo lo que nadie atendió. |
| **Pipes and Filters** | Etapas independientes que transforman un flujo de datos. | El pipeline general (descubrir → resolver → enriquecer → indexar). |
| **Bus de eventos de Forge/NeoForge** | Suscriptores con `EventPriority` (HIGHEST…LOWEST) y opción de ignorar eventos ya cancelados. | Prioridades conocidas por los modders; idea de “ya atendido”. |
| **Registro de drivers de `database/sql`** (Go) | Los drivers se registran en `init()` mediante import con efectos secundarios. | **Lo evitamos**: el registro global en `init()` complica las pruebas. Preferimos **registro explícito** en la raíz de composición. |

Conclusión: la arquitectura propuesta es un patrón probado. Es la misma idea
que usan los escáneres de software (Syft, Trivy), donde también hay muchas
fuentes heterogéneas, solapamientos y versiones.

## 2. Diseño

### 2.1 Piezas

```
                 ┌───────────────────────────────────────────┐
  ResourceIndex ─▶│            Puerta de descubrimiento        │
 (archivos del   │                                           │
  modpack ya     │  Fase 1 · Específicos  (alta confianza)    │
  fusionados)    │    structure-templates  (NBT + LootTable)  │
                 │    processor-lists      (append_loot)      │
                 │    vanilla-knowledge    (tabla por versión)│
                 │    lootr-extras         (élitros → cofre)  │
                 │  Fase 2 · Relacionales                     │
                 │    mob-spawns           (bioma → mob)      │
                 │    location-conditions  (location_check)   │
                 │    overrides            (config del usuario)│
                 │  Fase 3 · Heurísticos                      │
                 │    name-matching        (tabla ↔ estructura)│
                 │  Fase 4 · Genérico (siempre el último)     │
                 │    by-path              (chests/, entities/…)│
                 └──────────────┬────────────────────────────┘
                                ▼  Fuentes + Reclamaciones
                 ┌───────────────────────────────────────────┐
                 │        Puerta de enriquecimiento           │
                 │    lootr (por jugador, refresh, decay)     │
                 │    probabilities, localization …           │
                 └──────────────┬────────────────────────────┘
                                ▼
                            LootIndex
```

### 2.2 Contrato de un descubridor (borrador en Go)

```go
// Discoverer es una pieza enganchable que encuentra fuentes de loot.
type Discoverer interface {
    // Descriptor declara identidad, fase, prioridad y compatibilidad.
    Descriptor() Descriptor
    // Discover lee del contexto y publica fuentes en el registro de reclamaciones.
    Discover(ctx context.Context, in Input, out *Claims) error
}

type Descriptor struct {
    ID       string       // "structure-templates"
    Phase    Phase        // Specific, Relational, Heuristic, Generic
    Priority int          // orden dentro de la fase (mayor = antes)
    Applies  Applicability
}

type Applicability struct {
    Versions     VersionRange // ">=1.20.1 <1.21", vacío = todas
    Loaders      []Loader     // vacío = todos
    RequiresMods []string     // p. ej. "lootr"
}
```

- **Variantes por versión:** `lootr-extras@1.20` y `lootr-extras@26` son dos
  tipos distintos con el mismo `ID` y rangos de versión disjuntos. El
  registro elige la variante que aplica y comprueba al arrancar que no haya
  dos variantes solapadas (error de configuración, no de ejecución).
- **Orden:** por `Phase` y, dentro de la fase, por `Priority`; el desempate es
  el `ID`, para que el resultado sea determinista.
- **Genérico:** la fase `Generic` está reservada y el registro garantiza que
  se ejecuta la última.

### 2.3 Reclamaciones (*claims*): cómo se filtran las fuentes ya consideradas

Cada fuente descubierta se publica como una **reclamación**:

```
Claim{ LootTable, Kind (container/entity/fishing…), Owner (estructura, mob, bioma…),
       Confidence (exact/known/heuristic/manual), DiscoveredBy, Evidence (archivo, pieza…) }
```

Reglas:

1. Una loot table puede tener **muchas** reclamaciones legítimas (la misma
   tabla en varias estructuras).
2. Si dos descubridores reclaman la **misma pareja** (tabla, dueño), se
   conserva la de **mayor confianza** y se fusiona la evidencia (enfoque de
   Syft).
3. Los heurísticos **no reclaman** una tabla que ya tenga una reclamación
   `exact` o `known`.
4. El **genérico** solo actúa sobre las tablas **sin ninguna reclamación** y
   las clasifica por ruta (`chests/`, `entities/`, `gameplay/`, `fishing/`,
   `archaeology/`…) con confianza `unknown`.
5. `Claims` es la única forma de publicar resultados: los descubridores no
   se conocen entre sí.

### 2.4 Enriquecedores

Algunas piezas no **descubren** fuentes, sino que las **describen** (Lootr,
probabilidades, nombres). Van en una segunda puerta con el mismo mecanismo
de `Descriptor`/variantes, pero con un contrato distinto:

```go
type Enricher interface {
    Descriptor() Descriptor
    Enrich(ctx context.Context, in Input, sources *SourceSet) error
}
```

Separarlo cumple ISP: un descubridor no necesita saber anotar, y un
enriquecedor no puede crear fuentes nuevas.

### 2.5 Entrada compartida

`Input` da acceso de solo lectura a:

- `ResourceIndex`: archivos fusionados con la regla “el último gana”,
  consultables por tipo (`loot_table`, `structure`, `template_pool`…) **ya
  normalizados por versión**. Así los descubridores no conocen las rutas en
  plural o singular.
- `Target`: versión de MC, cargador y mods presentes.
- `ContainerCatalog`: IDs de bloques y entidades que llevan loot (vanilla,
  Lootr, otros mods); también es enganchable.
- `Diagnostics`: para informar sin abortar.

### 2.6 Registro explícito

```go
// raíz de composición (cmd/modpacklooter)
reg := discovery.NewRegistry(
    templates.New(), processors.New(), vanillaknowledge.New(),
    lootr.ExtrasV1_20{}, lootr.ExtrasV26{},
    spawns.New(), conditions.New(), overrides.New(cfg),
    heuristics.NameMatching{},
    generic.ByPath{},
)
plan := reg.Plan(target) // filtra por versión/cargador/mods y ordena
```

`plan` se puede imprimir (`modpacklooter scan --explain`) para ver qué piezas
se ejecutarán y en qué orden. Es útil para depurar las reviews.

## 2.7 Implementado (1.20.x)

| Puerta | Plugin | Fase | Aplica a |
|---|---|---|---|
| Descubrimiento | `structure-templates` | específica (100) | todas |
| Descubrimiento | `vanilla-knowledge` | específica (90) | `>=1.20 <1.21` |
| Descubrimiento | `lostcities` | específica (80) | mod `lostcities` |
| Descubrimiento | `name-matching` | heurística | todas |
| Descubrimiento | `generic-by-path` | genérica | todas |
| Enriquecimiento | `lootr` | específica | `>=1.20 <1.21` + mod `lootr` |

Todos se registran en `app/internal/plugins/plugins.go`. `modpacklooter plan`
muestra el orden para cualquier versión, cargador y lista de mods.

## 3. Cómo se aplica SOLID

| Principio | En la puerta de descubrimiento |
|---|---|
| S | Una pieza = una fuente. |
| O | Soportar una fuente o una versión nueva = añadir una pieza registrada. |
| L | Todas las variantes cumplen el mismo contrato y pasan la misma batería de tests. |
| I | `Discoverer` y `Enricher` están separados; `Input` expone vistas de solo lectura. |
| D | El núcleo depende de las interfaces; el registro concreto vive en la raíz de composición. |

## 4. Pruebas

- **Tests por pieza** con un `ResourceIndex` en memoria.
- **Test del registro**: orden determinista, el genérico siempre último y
  ninguna variante solapada para cualquier versión del rango soportado.
- **Test de reclamaciones**: deduplicación por confianza y que el genérico
  no reclame tablas ya reclamadas.

## Fuentes

- [Syft: ownership-by-file-overlap](https://anchorecommunity.discourse.group/t/revisiting-ownership-by-file-overlap-relationships/137) · [paquete `cataloging`](https://pkg.go.dev/github.com/anchore/syft/syft/cataloging)
- [Trivy fanal (resumen)](https://factory.ai/open-source-wikis/trivy?page=systems%2Ffanal.md) · [paquete `analyzer`](https://pkg.go.dev/github.com/deepfactor-io/trivy/v3/pkg/fanal/analyzer)
- [Caddy: Extending Caddy](https://caddyserver.com/docs/extending-caddy) · [`httpcaddyfile` (orden de directivas)](https://pkg.go.dev/github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile)
- Código de Lootr como ejemplo real de MultiLoader (`common/` + `neoforge/` + `fabric/`): [LootrMinecraft/Lootr](https://github.com/LootrMinecraft/Lootr)
