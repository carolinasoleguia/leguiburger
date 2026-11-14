---
name: sast-configuration
description: "Preparar o revisar analisis estatico y de dependencias Python/TypeScript de Al Toque con evidencia verificable."
---

# Análisis de seguridad del código

## Alcance y selección

Usar al preparar, configurar o ejecutar un escáner de seguridad/dependencias.
Cubre herramientas y evidencia de análisis; revisión manual de permisos o
amenazas corresponde a `security-review` y no necesita un escáner automáticamente.

Leer el [contexto del repo](../../../../AGENTS.md), locks de dependencias y
`.github/workflows/verify.yml`. Esta skill no incluye ni instala un escáner.

- Seleccionar herramienta según el objetivo y disponibilidad: patrones/taint
  de Python/TypeScript, secretos o vulnerabilidades de dependencias bloqueadas.
  Verificar documentación actual, soporte de Windows/CI y licencias de uso.
  Separar SAST, secretos y dependencias: ninguna familia comprueba por sí sola
  permisos efectivos, lógica comercial o resistencia del modelo.
- Priorizar flujos propios: service role en cliente, autorización con metadata
  editable, RPC/RLS excesivos, SSRF/URLs externas, uploads ilimitados, firmas
  omitidas y entrada no confiable en instrucciones/herramientas del modelo.
- Conservar separación entre clave anon/publicable y secreto. No imprimir
  valores de credenciales en findings, artefactos o logs del escáner.
- Seguir validación/escape según destino: SQL parametrizado, XML sin entidades
  externas, texto HTML/PDF escapado y archivos con límites/formatos verificados.
  La sanitización genérica no sustituye esos controles ni la autorización.
- Excluir dependencias/generated/builds y las entradas de skills para que
  ejemplos de instrucciones no aparezcan como código de aplicación.
- Validar hallazgos en consumidores/flujo y conservar tests relevantes.
  Documentar falsos positivos; no suprimir por carpeta para obtener verde.
  Conservar ID/regla, archivo, base/commit, versión de herramienta y estado
  (abierto, verificado, resuelto, no aplica) para comparar otra ejecución.
  No tomar fallo de ejecución/parseo, análisis omitido o entorno vacío como cero
  vulnerabilidades. Auditar el entorno activo no demuestra análisis de locks/
  images del deploy. Verificar egress, redacción, validación remota de secretos,
  consultas de advisories y telemetría de la herramienta elegida.
- Incorporar un gate a CI solo si está dentro del alcance: establecer baseline
  y gravedad antes de bloquear todos los cambios por deuda preexistente.

Reportar herramienta/versiones, comandos, alcance y salida real. Si solo se
revisaron archivos, llamarlo revisión manual. No afirmar auditoría completa,
cumplimiento normativo ni rotación de credenciales sin evidencia.
