# Implementación y tipos Python

Los nombres de archivo son relativos a la raíz del repo. Partir del módulo y
sus consumidores, no de un layout nuevo para toda la aplicación.

- Mantener `apps/api/app/{api,core,bot,infrastructure}`, `apps/worker/worker`
  y `packages/contracts/src/altoque_contracts`; rutas para transporte, casos
  de uso para reglas/permisos y adaptadores para I/O.
- El worker no importa archivos de API para duplicar reglas. Cada deploy debe
  contener sus contratos/dependencias sin importar archivos de otro deploy.
- Extraer funciones/componentes para responsabilidades o decisiones reales.
  Usar composición y dependencias sustituibles con el patrón existente;
  introducir Protocol/interfaz cuando haya una sustitución o consumidor útil.
- En un refactor, mantener unidad de consistencia, efectos, permisos y
  contratos. Separar cambios de comportamiento del movimiento de código.
  Conservar journals/conciliación; no esconder estado fiscal en objetos locales.
- Al mover módulos, revisar imports, patches de tests, entrypoints y builds
  desde cada proceso. Evitar ciclos e I/O de red al importar.
- Para Python 3.11 usar `TypeVar`, `Protocol`, `TypedDict`, uniones y colecciones
  parametrizadas compatibles. No copiar alias/generics exclusivos de 3.12+.
- Separar JSON recibido de DTO validado: una anotación o `cast` no valida
  payloads, identidad ni permisos. Representar ausencia/éxito/fallo/incertidumbre
  sin perder información del dominio en un bool o dict genérico.
- Reutilizar validadores de documentos, teléfonos y fechas. Evitar estado global
  mutable o argumentos mutables que mezclen solicitudes/talleres/tests.
- Documentar contratos no obvios, no lo que repite la firma. Mantener estilo,
  nombres e imports locales y limitar el diff a los archivos del cambio.
- No corregir un fallo de proveedor con un catch amplio que devuelve éxito o
  una colección vacía. Transportes y trabajos durables tienen su procedimiento
  en `python-reliability`.

Verificar consumidores y regresiones del módulo. Para checks estáticos, usar
herramientas configuradas y describir evidencia real; no habilitar strict o
formatting global como condición para una edición ordinaria.
