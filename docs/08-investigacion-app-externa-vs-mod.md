# 08 · Investigación: ¿app externa o mod multicargador?

> Fecha: 2026-10-02. Objetivo: decidir si es más fácil soportar **1.20.1 →
> 26.3** y **Forge + NeoForge (Fabric a futuro)** desde una app externa o
> desde un mod. Decisión resultante: [ADR-0004](decisiones/0004-app-externa-con-volcador-opcional.md).

## 1. Contexto del ecosistema (2026)

| Hecho | Impacto |
|---|---|
| Desde 2026 Minecraft numera por año: **26.1** (mar-2026), 26.2, **26.3** (la más reciente). | Rango objetivo: 1.20.1 … 1.21.x … 26.1 … 26.3. |
| **26.1 se publica sin ofuscar**; Fabric deja Yarn y todos pasan a nombres de Mojang. | Un mod que cubra 1.20.1 y 26.x convive con dos mundos de mappings. |
| Java requerido: **17** (1.20.1–1.20.4), **21** (1.20.5–1.21.x), **25** (26.x). | Un mod necesita tres toolchains; una app externa, ninguna. |
| Forge sigue activo (p. ej. Forge 62.x para 26.1); NeoForge también. | Los dos cargadores objetivo existen en todo el rango. |
| NeoForge en 1.20.1 es un *fork* casi idéntico de Forge 47.1; desde 1.20.2 diverge. | En 1.20.1 los dos se tratan casi igual. |
| 26.3 cambia las loot tables: `rolls`/`bonus_rolls` aceptan un ID de *number provider*, y `set_loot_table` renombra `name` a `loot_table_id`. | Los **formatos de datos** sí cambian, pero se documentan en los changelogs y son JSON. |

## 2. Cómo se construyen los mods multicargador

### 2.1 Multicargador (un código, varios cargadores)

| Enfoque | Cómo funciona | Pros | Contras |
|---|---|---|---|
| **MultiLoader-Template** (jaredlll08) | Proyecto Gradle con módulo `common` compilado contra vanilla y módulos `forge`/`neoforge`/`fabric` que lo incluyen y aportan los puntos de entrada. Las diferencias se resuelven con interfaces + `ServiceLoader`. | Sin dependencias en tiempo de ejecución; es “un estilo de código”. | Hay que escribir cada abstracción de plataforma. |
| **Architectury (Loom + API)** | Plugin Gradle que remapea un `common` para cada plataforma; `@ExpectPlatform` para código específico; API con abstracciones (eventos, registros). | Mucho resuelto ya. | Dependencia extra (Architectury API) en el modpack; plugin grande y opaco. |
| **Sinytra Connector** | Ejecuta mods de Fabric sobre NeoForge. | — | No es una arquitectura de desarrollo; no aplica. |

Ambos enfoques son, en el fondo, **puertos y adaptadores**: `common` = núcleo,
módulos de cargador = adaptadores.

### 2.2 Multiversión (un código, varias versiones de Minecraft)

| Herramienta | Cómo funciona |
|---|---|
| **Stonecutter** | Plugin Gradle que genera un subproyecto por versión y activa/desactiva bloques de código con **comentarios condicionales** (preprocesador), p. ej. `//? if >=1.21 {`. |
| **Stonecraft**, `multiloader-plugin`, plantillas `stonecutter-mod-template` | Combinan Stonecutter (versiones) + Architectury o MultiLoader (cargadores) para obtener una matriz *versión × cargador* desde un repo. |
| Ramas por versión | Alternativa clásica: una rama git por versión, *cherry-pick* de cambios. Mucho mantenimiento. |

### 2.3 Lo que costaría un mod para este proyecto

Matriz mínima de artefactos a compilar, probar y publicar:

```
1.20.1  × {Forge, NeoForge(fork)}      Java 17, SRG/Mojmap
1.21.1  × {Forge, NeoForge}            Java 21, componentes de item, carpetas en singular
26.1–26.3 × {Forge, NeoForge}          Java 25, sin ofuscar, cambios en loot/number providers
(+ intermedias que pida la comunidad, + Fabric en el futuro)
```

APIs internas que el mod tocaría y que cambiaron dentro del rango:
cómo se accede a las loot tables recargables (cambió en 1.21.x), los *codecs*
de loot, `HolderSet`/registros de biomas, eventos de cada cargador
(`LootTableLoadEvent`, GLM). Cada cambio supone un bloque condicional de
Stonecutter y una prueba **arrancando el juego o un servidor** con el modpack.

## 3. Cómo se vería desde una app externa

La app externa solo lee **datos**: JSON de `data/`, plantillas `.nbt`, `lang`
y texturas dentro de los `.jar`. No ejecuta código de Minecraft.

| Aspecto | App externa | Mod |
|---|---|---|
| Ofuscación / mappings | **No le afecta.** Los datos nunca estuvieron ofuscados. | Afecta (SRG → Mojmap → sin ofuscar). |
| Versión de Java | Ninguna (binario nativo). | 17 / 21 / 25. |
| Añadir una versión de MC | Añadir/ajustar un **adaptador de formato** según el changelog de datapacks. | Nuevo objetivo de compilación + bloques condicionales + pruebas en juego. |
| Añadir un cargador | Leer otro archivo de metadatos (`mods.toml`, `neoforge.mods.toml`, `fabric.mod.json`) y otras rutas (GLM, biome modifiers). | Otro módulo de plataforma con sus eventos. |
| Necesita arrancar el juego | No. Segundos sobre la carpeta del modpack. | Sí, con todo el modpack cargado (minutos y mucha RAM). |
| Publicar en GitHub/GitLab Pages desde CI | Trivial: el binario corre en CI. | Requiere levantar un servidor en CI. |
| Precisión | Alta para lo declarado en datos; **no ve** loot asignado por código o por scripts. | Ve el estado final ya resuelto (datapacks, KubeJS, eventos). |
| Estructuras vanilla “por código” (mineshaft, fortaleza, stronghold…) | Necesita una tabla de conocimiento incluida por versión. | Igual: tampoco están en datos; habría que mapearlas a mano. |

### Conclusión

**Soportar varias versiones y cargadores es claramente más fácil desde una app
externa**: el problema pasa de *compatibilidad binaria con el juego* (matriz de
compilación, mappings, Java, APIs internas) a *compatibilidad de formatos de
datos*, que son JSON/NBT documentados, cambian poco y se prueban con fixtures.

La única ventaja real del mod es ver el **estado resuelto en tiempo de
ejecución**. Eso se puede recuperar más adelante con un mod **mínimo** que
solo **vuelca** registros (loot tables, estructuras, biomas, tags) a JSON con
los propios *codecs* del juego. El resultado tiene el mismo formato que un
datapack, así que la app lo leería con los mismos parsers, como una fuente más
(`ModpackSource` → `RuntimeDumpSource`).

El mod volcador tendría una superficie de API muy pequeña, lo que lo hace
viable con MultiLoader-Template + Stonecutter cuando llegue el momento.

## 4. Hallazgos útiles para el diseño

- **`append_loot` en processor lists** (`worldgen/processor_list`): las
  estructuras jigsaw pueden asignar loot a cofres al generarse mediante un
  procesador `minecraft:rule` con `block_entity_modifier` de tipo
  `minecraft:append_loot`. Es dato estático y hay que leerlo, no solo el NBT.
- **Data markers**: algunas estructuras (p. ej. naufragios) colocan bloques de
  estructura con un texto (`Metadata`) y el código decide la tabla. Se
  resuelven con la tabla de conocimiento + overrides.
- **Entidades con loot**: los cofres en vagonetas (`chest_minecart`) de las
  minas se colocan por código; en plantillas NBT pueden aparecer como
  entidades con `LootTable`.
- **Mobs → bioma**: `spawners` en el JSON del bioma + *biome modifiers* de
  Forge/NeoForge (`add_spawns`) permiten asociar el loot de entidades a biomas.
- **Condiciones `location_check`** con bioma o estructura dentro de una loot
  table (p. ej. pesca en la jungla) también asocian loot a biomas.
- **misode/mcmeta** publica los datos vanilla de cada versión en git, y
  `misode.github.io/versions` tiene changelogs técnicos: buenas fuentes para
  fixtures y para detectar cambios de formato.

## Fuentes

- [Minecraft Java Edition 26.3 (minecraft.net)](https://www.minecraft.net/en-us/article/minecraft-java-edition-26-3)
- [Java Edition 26.3 (minecraft.wiki)](https://minecraft.wiki/w/Java_Edition_26.3)
- [What Breaks in Minecraft 26.3 (SyntaxMine)](https://syntaxmine.com/articles/what-breaks-in-minecraft-26-3)
- [Removing obfuscation in Java Edition (minecraft.net)](https://www.minecraft.net/en-us/article/removing-obfuscation-in-java-edition)
- [Migrating Mappings (Fabric docs)](https://docs.fabricmc.net/develop/porting/mappings/)
- [NeoForge for Minecraft 26.1](https://neoforged.net/news/26.1release/)
- [Forge para 26.1](https://files.minecraftforge.net/net/minecraftforge/forge/index_26.1.html)
- [Java 25 para 26.x (mc-node.net)](https://mc-node.net/blog/en/which-java-version-minecraft-server/)
- [MultiLoader-Template (DeepWiki)](https://deepwiki.com/jaredlll08/MultiLoader-Template)
- [Architectury](https://architectury.org/)
- [Stonecraft](https://github.com/meza/Stonecraft) · [stonecutter-mod-template](https://github.com/rotgruengelb/stonecutter-mod-template) · [multiloader-plugin](https://github.com/BizCub/multiloader-plugin)
- [Processor list (minecraft.wiki)](https://minecraft.wiki/w/Processor_list)
- [Minecraft 26.1 changelog (misode)](https://misode.github.io/versions/?id=26.1&tab=changelog)
- Mods que ya muestran loot dentro del juego: [Just Enough Resources](https://modrinth.com/mod/just-enough-resources-jer) y [Simple Loot Viewer](https://modrinth.com/mod/simple-loot-viewer/version/vGGd59JN)
