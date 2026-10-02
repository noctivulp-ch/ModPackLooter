# ADR-0007: Distribución por GitHub Releases y sitio compatible con GitHub/GitLab Pages

- **Estado:** Aceptada
- **Fecha:** 2026-10-02

## Contexto

La app se usará en el PC del usuario y, potencialmente, en CI. Los sitios
generados se quieren publicar en GitHub Pages o GitLab Pages.

## Decisión

- La app se distribuye como **binarios autocontenidos** en GitHub Releases
  (GoReleaser en GitHub Actions) y se puede compilar localmente con un comando.
- El sitio generado usa **solo enlaces relativos**, una carpeta con
  `index.html` por página y `.nojekyll`, de forma que funciona en
  `https://<usuario>.github.io/<repo>/`, en GitLab Pages y desde `file://`.
- La CLI ofrecerá plantillas de CI para regenerar y publicar el sitio.

## Consecuencias

- No se permiten rutas absolutas (`/objetos/...`) en las plantillas; habrá un
  test que lo verifique.
- El usuario no necesita instalar runtimes.
