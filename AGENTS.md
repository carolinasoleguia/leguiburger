# LeguiBurger

El frontend usa Vue 3 y Vite en `frontend/`; el backend usa Go en
`internal/`, `pkg/` y `cmd/`. Conservar este stack y las convenciones existentes.

## Skills del repositorio

La biblioteca importada de Al Toque está en [skills/README.md](skills/README.md).
Antes de aplicar una guía, leer el [adaptador de LeguiBurger](skills/leguiburger-context.md).
Elegir el menor conjunto que cubra el encargo y leer sus `SKILL.md` completos.
Las referencias a Al Toque, Next o FastAPI describen el producto de origen;
no cambian las tecnologías ni los permisos de este proyecto.

Para frontend, comenzar por [web-ui-design](skills/frontend/web-ui-design/SKILL.md).
Para el pulido visual solicitado, [unslop](skills/frontend/review/unslop/SKILL.md)
incluye sus referencias, scripts y atribución original.

Validar con `npm --prefix frontend run build`, `python tools/check-source.py`
y las pruebas de navegador de `tools/` pertinentes al cambio. La integridad
de la biblioteca se verifica con `python tools/check-skills.py`.
No instalar otro framework ni ejecutar instaladores o helpers por cargar una skill.
