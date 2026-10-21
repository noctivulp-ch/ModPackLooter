<div align="center">

# ModPackLooter

**Todas las fuentes de botín de un modpack de Minecraft, en un sitio web estático.**

![Licencia](https://img.shields.io/badge/licencia-GPL--3.0--or--later-FF6B00?style=flat-square&labelColor=1A1A1A)
![Plataformas](https://img.shields.io/badge/plataformas-Linux%20%C2%B7%20Windows%20%C2%B7%20macOS-3A3A3A?style=flat-square&labelColor=1A1A1A)
![Go](https://img.shields.io/badge/Go-1.24-3A3A3A?style=flat-square&logo=go&labelColor=1A1A1A)

</div>

---

## Qué es

ModPackLooter lee un modpack de Minecraft (mods, datapacks y configuración) y genera un sitio web
que responde dónde conseguir cada cosa: en qué estructuras y biomas aparece cada objeto, qué
contiene cada estructura, qué deja cada criatura y qué se puede pescar o comerciar. El
resultado es HTML estático: se abre en el navegador o se publica en cualquier alojamiento.

## Funciones principales

- Tablas de botín de estructuras, criaturas, pesca y comercio con aldeanos.
- Probabilidades por fuente, con resolución de *tags* y de variantes de objeto (libros
  encantados, pociones, NBT de mods).
- Fichas por objeto, estructura, bioma y criatura, enlazadas entre sí.
- Búsqueda, incluida una búsqueda avanzada que propone los lugares que reúnen una lista de
  objetos.
- Compatibilidad con Lost Cities y Lootr.

## Tecnologías

| Capa | Tecnología |
|---|---|
| Programa | Go, CLI con Cobra |
| Lectura | ZIP/JAR, JSON, NBT y TOML |
| Sitio | plantillas de Go, HTML, CSS y JavaScript sin dependencias |
| Entrega | binarios autocontenidos para Linux, Windows y macOS |

## Puesta en marcha

Descarga el binario de tu sistema desde las *releases*, o compílalo:

```bash
cd app
go build -o modpacklooter ./cmd/modpacklooter
./modpacklooter --help
```

| Carpeta | Contenido |
|---|---|
| [`app/`](app/) | Programa principal |
| [`mod/`](mod/) | Futuro mod que exportará los datos que no se pueden leer de los archivos |
| [`docs/`](docs/README.md) | Visión, arquitectura, investigación y decisiones (ADR) |

> Las texturas e íconos que aparezcan en un sitio generado pertenecen a sus autores: Mojang y
> los autores de cada mod. Quien publica un sitio es responsable de respetar sus licencias.

## Licencia

Código bajo **GPL-3.0-or-later** ([`LICENSE`](LICENSE)). Los componentes de terceros y sus licencias
están en [`NOTICE`](NOTICE). El nombre y la identidad visual no forman parte de la licencia: ver
[`TRADEMARKS.md`](TRADEMARKS.md). Para aportar, [`CONTRIBUTING.md`](CONTRIBUTING.md).

## Apoyo

Este proyecto forma parte del trabajo de **NoctiVulp**: herramientas para Linux y Android,
publicadas como software libre. Las compilaciones oficiales y las novedades salen en Patreon.

<p align="center">
  <a href="https://www.patreon.com/cw/NoctiVulp"><img src="https://img.shields.io/badge/Patreon-NoctiVulp-FF6B00?style=for-the-badge&logo=patreon&logoColor=white&labelColor=1A1A1A" alt="NoctiVulp en Patreon"></a>
</p>

---

<p align="center"><sub>© 2026 NoctiVulp · <b>Powered by AI</b> — desarrollado con la asistencia de Claude, de Anthropic.</sub></p>
