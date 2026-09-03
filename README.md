# fountain — CLI Go `.fountain` → PDF

Convierte guiones en formato Fountain a PDF con portada profesional.

```bash
go run . script.fountain -o script.pdf
go run . -o out.pdf script.fountain
```

## Frontmatter YAML

Bloque inicial opcional entre `---`. Soporta `kebab-case`, `snake_case` o `con espacios` (`revision-color = revision_color = Revision Color`). Listas con `- ` se unen con `\n`. Claves desconocidas van a `Custom` y también se renderizan.

| Key | Zona portada |
|-----|--------------|
| `title`, `credit`, `author`, `adapted-by`, `based-on`, `episode-title` | Arriba-centro |
| `logline`, `tagline`, `genre`, `subgenre`, `tone`, `format`, `season`, `episode`, `runtime`, `language`, `comps`, `themes`, `status`, `production-company`, `producer`, `director-attached`, `budget-level`, `source`, `revision` | Medio-centro |
| `draft-type`, `version`, `revision-color`, `revision-date`, `draft-date`, `pages`, `contact`, `address`, `email`, `phone`, `agent`, `manager`, `copyright`, `wga-registration`, `rights` | Pie |
| `synopsis`, `dedication`, `disclaimer` | Página 2 |

`revision-color`: `white | blue | pink | yellow | green | goldenrod`.

## Ejemplo completo

```yaml
---
title: LA SEÑAL
credit: Escrito por
author:
  - Sarahi Zepeda Carrera
  - Miguel Hernandez Moreno
draft-date: 2026-09-03
contact: Moonlight Studios
source: Historia original
revision: Borrador 3
logline: Una operadora detecta una señal imposible y debe decidir si responder.
tagline: El silencio también habla.
synopsis: Elena capta una señal que no debería existir y arriesga todo por responder.
genre: Ciencia ficción
subgenre: Thriller
tone: Melancólico
themes:
  - memoria
  - primer contacto
comps:
  - Arrival
  - Contact
format: series
season: 1
episode: 2
episode-title: La frecuencia perdida
runtime: 52 min
pages: 58
language: es-MX
status: Primer borrador
production-company: Moonlight Studios
producer: Ana Ruiz
director-attached: L. Méndez
budget-level: Bajo
adapted-by: Adaptado por J. Doe
based-on: Idea original de S. Zepeda
copyright: © 2026 Moonlight Studios
wga-registration: WGA 123456
rights: Todos los derechos reservados.
dedication: Para los que miran el cielo.
disclaimer: Obra de ficción.
draft-type: Primer borrador
version: v1.0
revision-color: blue
revision-date: 2026-09-03
email: contacto@moonlight.example
phone: +52 55 1234 5678
agent: Agencia Norte
manager: M. Soto
address:
  1234 Sunset Blvd
  Los Angeles, CA 90028
mi-campo-custom: va a Custom y también se renderiza
---

EXT. ESPACIO - NOCHE

Estrellas cubren el vacío.
```

Ver fixture funcional en `testdata/extended-frontmatter.fountain`.
