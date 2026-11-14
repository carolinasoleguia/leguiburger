# Configuración de servicios

Leer el [contexto del repo](../../../../AGENTS.md),
`apps/api/app/core/config.py`, ejemplos `.env.example` de cada aplicación y
`docker-compose.yml`. La API usa una dataclass de Settings y `from_env`.

- Añadir variables al contrato del servicio que las consume; documentar
  obligatoriedad, default, formato y validación al arrancar/usar el adaptador.
- Distinguir `NEXT_PUBLIC_API_URL` accesible al navegador de `ALTOQUE_API_URL`
  del servidor. Una variable pública de build requiere rebuild al cambiar.
- Mantener secretos fuera de argumentos de build y del browser. El worker
  recibe infraestructura, no credenciales de proveedores que solo usa la API.
- Conservar defaults y fallos cerrados de firma/secreto en producción. No
  habilitar features sensibles para que una fixture pase.
- Settings se instancia al importar: en tests aplicar entorno antes de importar
  o sustituir la dependencia en el consumidor, restaurando estado después.
- Mantener compatibilidad API/web en claves de cifrado y secretos internos;
  una migración de configuración no implica rotarlos automáticamente.

Probar ausencia, valor inválido y configuración válida con valores ficticios.
No incorporar otra librería de Settings para una nueva variable. Revisar
`docs/operacion/despliegue.md` si el cambio afecta build, runtime o corte coordinado.
