# AI Toque: contrato de ejecución de la biblioteca de seguridad

AI Toque es la marca de la biblioteca, que puede servir a distintos productos.
Al Toque es el producto de este checkout. Los perfiles conservan especialidades,
procedimientos, checklists y ejemplos amplios; su disponibilidad no se recorta
porque una tecnología o riesgo no exista en el producto actual.

Esta capa adapta la ejecución y tiene precedencia sobre supuestos y reglas
genéricas de los perfiles originales. No reemplaza su contenido especializado.
Cada invocación debe leer el perfil completo y seleccionar aplicabilidad por
evidencia. No resumirlo para decidir de antemano qué podrá necesitar otro producto.

## 1. Contexto y alcance de cada ejecución

Identificar producto/repo, versión/base/diff, objetivo, stack/versiones declarados,
servicios, datos, ambientes y autorización vigente. Leer instrucciones del producto
si existen; si faltan, construir el inventario desde código/config/locks/docs y
declarar incertidumbres. La biblioteca no depende de rutas locales de una máquina.

Para Al Toque, el adaptador es `AGENTS.md` de la raíz: Next conserva presentación,
Auth y transporte; FastAPI concentra negocio/proveedores; worker coordina outbox;
Supabase aporta Auth/SQL/Storage. Python objetivo 3.11. No mover proveedores a
Next, duplicar reglas en worker o introducir hashing/passwords propios al adaptar
un ejemplo. Otro producto utiliza su propia arquitectura y versión de Python.

Ejemplos Express/Node, Java, cloud, Kubernetes, GraphQL, WebSocket, Solidity o SIEM
se conservan. Aplicarlos cuando existan esas superficies; adaptar principios al
framework real sin instalar uno distinto para reproducir un ejemplo.

## 2. Aplicabilidad, cobertura y evidencia

Por capacidad/control relevante registrar uno de estos estados:

| Estado | Significado | Evidencia requerida |
| --- | --- | --- |
| Aplicable, verificado | La superficie existe y se comprobó el control/caso | Archivo/flujo, comando/test o evidencia autorizada y versión/ambiente. |
| Aplicable, hallazgo | La superficie existe y hay defecto o riesgo sustentado | Precondición, dato/actor, camino al efecto, impacto y corrección/aceptación. |
| No aplicable en este alcance | La superficie no existe o no interviene aquí | Razón específica e inventario que la sustenta; la capacidad sigue disponible. |
| Pendiente/no ejecutado | Es pertinente pero falta dato, herramienta o acceso | Qué falta y qué prueba permitiría verificar; no presentarlo como aprobado. |

No hay una clasificación global de “repo seguro” por ausencia de alertas.
Separar control observado, impacto inferido, ejecución local y estado remoto.
Un scanner faltante no vuelve no aplicable el análisis de dependencias existente.
Una policy o permiso en un archivo no acredita su despliegue efectivo.

## 3. Herramientas y autonomía

Los perfiles son instrucciones de especialistas, no instalaciones de scanners,
conectores, SIEM, cloud accounts o entornos de explotación. Inventariar herramientas
disponibles y verificar versión, flags, formatos, egress, redacción y permisos
antes de ejecutarlas. Los bloques de código son ejemplos completos conservados;
no son comandos autorizados, artefactos ya instalados ni pipelines ya existentes.

Una revisión autoriza lectura/análisis local proporcionado y pruebas locales sin
efectos externos. Instalaciones, scanners con red o cambios en CI deben corresponder
al trabajo solicitado y permisos del entorno. Explotación, escaneo activo remoto,
cobros, facturas, mensajes, rotación, contención y cambios de cuentas requieren
alcance concreto. Si ya está autorizado, proceder con sus comprobaciones; no
reabrir aprobaciones por una regla genérica del perfil.

Propuestas de colaboración entre especialistas no crean subagentes automáticamente.
Seleccionar el perfil por decisión; delegar solo cuando esté autorizado. No cargar
o encadenar los 11 especialistas completos por cada label o refactor rutinario.
Una auditoría amplia puede evaluar las 11 especialidades y ejecutar las aplicables.

## 4. Fidelidad al contenido y límites del rol

Conservar estructura, misiones, ejemplos, procedimientos, capacidades avanzadas
y criterios de evaluación. Los rasgos, “experiencia” y “memoria” del perfil describen
un especialista ideal; no son experiencia real del modelo, memoria persistente,
hechos de una auditoría previa ni herramientas que posea este agente.

Tiempos/SLA, porcentajes y severidades de ejemplos son políticas orientativas,
no obligaciones comerciales de AI Toque ni veredictos por una expresión regular.
Un reporte “clean” de ejemplo no se copia como resultado si no se ejecutó su
análisis. Escenario plausible y comprobación local pueden sustentar un hallazgo
sin explotar infraestructura; la duda se reporta con su límite, no se oculta.

## 5. Identidad y autorización

Verificar identidad con el mecanismo real y permiso por actor, ámbito, objeto,
acción, estado y vigencia. Metadata editable, IDs recibidos o headers libres no
conceden autoridad. Claims de servidor tampoco reemplazan pertenencia del objeto
o membresía comercial. Un IdP no tiene por qué ser fuente única de todos los roles.

Bearer, cookies, OAuth, API keys y capacidades públicas se evalúan por canal.
Supabase Auth del producto puede validar JWT sin el PyJWT del ejemplo. No exigir
HttpOnly como única arquitectura cuando existe un acceso Bearer directo diseñado;
revisar XSS, manejo de sesión/token y CSRF según el mecanismo efectivo.

## 6. Datos, RLS y recursos públicos

RLS habilitado no acredita aislamiento. Revisar grants, rol, operación, policies
combinadas, USING/WITH CHECK, owner/search_path de SECURITY DEFINER y scope del
service role. `USING (true)`, bucket público o UUID compartido puede representar
un recurso público intencional: comprobar proyección, permiso, vigencia y revocación.

No copiar `auth.uid() = user_id` universalmente a un modelo por taller/equipo.
Probar actor legítimo, otro ámbito, rol insuficiente y pérdida de autoridad. Usar
SQL/Storage efectivos de staging solo con el acceso y alcance autorizados.

## 7. Secretos, cifrado y rotación

No exponer secretos a cliente, código público, logs, URLs, modelo o artefactos.
Clasificar claves anon/publicables y fixtures separadamente de credenciales privadas.
Variables privadas de runtime pueden ser el mecanismo válido del hosting; no
exigir un vault ni cifrado adicional de toda env var por una frase importada.

Mantener capacidades de broker, claves dinámicas, KMS y rotación para plataformas
que las soporten. Elegir por propósito/consumidores y riesgo real. Borrar código
no revoca una clave; confirmar estado del proveedor. Preservar continuidad de
descifrado, callback OAuth y jobs al cortar una credencial. Purga de historia y
revocación inmediata no son acciones automáticas de una revisión.

## 8. Criptografía, red y disponibilidad

Utilizar primitivas/librerías verificadas, nonces adecuados y fallo cerrado según
el efecto. TLS, mTLS, identidad de workload, firewall/WAF y segmentación se evalúan
por topología real, sin suponer que existen o imponer una plataforma nueva.
Headers HTTPS como HSTS pueden estar en el borde; verificar evidencia efectiva.

Default deny en permisos no obliga a bloquear toda capacidad opcional caída.
Determinar qué requiere cada servicio/operación y su política de degradación.
Rate limiting, límites de recursos, replay, idempotencia y journal se complementan;
un timeout no acredita ausencia del efecto fiscal/financiero o mensaje externo.

## 9. IA y contexto

Separar instrucciones confiables de usuarios, referencias, archivos y resultados.
Un rol user o ausencia de tools no garantiza resistencia a prompt injection o
confidencialidad. Exponer tools no fija por sí solo severidad: revisar datos,
capacidad efectiva, identidad, ejecutor, destino y confirmación.

El ejecutor comprueba permisos/estado/argumentos fuera de la decisión del modelo.
Aplicar budgets y minimizar contexto antes de exportar; filtro posterior no retira
datos ya enviados. Pruebas con mocks verifican el ejecutor; evaluaciones de modelos
usan muestras/variantes autorizadas y no prueban invulnerabilidad.

## 10. Supply chain y código de ejemplos

Conservar análisis de dependencias, imágenes, firmas, SBOM, integridad y CI.
Un lock o action instalada no garantiza ausencia de CVE; verificar versión/alcance
y advisories primarios actuales. Tags/versions de ejemplos pueden haber cambiado.
No adoptar suppressions, CVE inventados, thresholds o `audit fix --force` como
política del producto. Un fallo del scanner o JSON inválido es cobertura incompleta.

## 11. Detección, inteligencia e incidentes

Mantener capacidades SIEM, Sigma, YARA, feeds y forense. Al ejecutarlas establecer
fuente de datos, herramientas, permisos y estado de la evidencia. Si el producto
no tiene SIEM, su plantilla no demuestra una detección desplegada; si falta acceso
a logs pertinentes, marcar pendiente. No inferir atribución/intrusión de un 403.

No fabricar IOCs, estadísticas o memoria de incidentes. Contención, avisos externos,
shares de inteligencia y análisis de sistemas de terceros tienen alcance propio.
No restaurar/reiniciar/purgar sistemas por un playbook sin esa autorización.

## 12. Compliance y auditoría

Conservar SOC 2, ISO, GDPR, HIPAA, PCI y demás capacidades del perfil, seleccionando
framework, jurisdicción, datos y período realmente aplicables. No inferir que todo
producto de AI Toque trata PHI/CHD o necesita cada certificación. Evidencia de código
o una plantilla de controles no acredita certificación ni cumplimiento legal.

## 13. Pentest y tecnologías del producto

El perfil de pentest permanece completo. Definir targets, redes/cuentas,
versión/deploy, reglas de engagement, ventanas, datos ficticios y límites
antes de actividad ofensiva o con efectos económicos. Blockchain fue
excluido por el usuario porque no forma parte de los productos AI Toque;
su skill y snapshots ya no forman parte de esta biblioteca.

## 14. Pruebas y ejemplos

Adaptar ejemplos al framework/versiones existentes. Conservar tests y herramientas
de cada producto; no migrar unittest a otro runner o cambiar Auth para aplicar
el ejemplo. Registrar fixtures sintéticas y resultados locales separadamente de
proveedores, grants SQL, hosting y redes efectivas.

## 15. Informe y verificación posterior

Cada finding tiene identidad estable, origen/regla, ubicación, base/commit,
precondición, impacto, certeza, corrección, caso de aceptación y evidencia de cierre.
Comparar abierto/resuelto/nuevo contra la misma superficie y reglas. Cambiar un
path o desactivar una regla no demuestra cierre. Separar fix local de revocación,
policy aplicada o deployment real. Preservar evidencia saneada y cobertura pendiente.

## 16. Estándar del producto y precedencia

Si un perfil menciona `security/17-security-pattern.md`, buscar el estándar real
del producto: no fue aportado con los perfiles originales. En su ausencia usar
este contrato común y fuentes primarias, sin inventar contenido o secciones del
documento ausente. Los números § de sus ejemplos no son requisitos comprobados.
Si el producto aporta ese estándar, usar su versión y registrar discrepancias.

Las instrucciones del usuario/producto y permisos vigentes gobiernan el alcance.
En conflictos técnicos priorizar la garantía requerida y evidencia, documentando
tradeoffs. Este contrato no convierte los textos originales en permiso de operar.

## 17. Preparación y trazabilidad para un auditor de AI Toque

Antes de reportar un audit, registrar producto/base, perfiles seleccionados,
controles aplicables/N/A/pendientes, herramienta/config/versión, comandos y fechas,
resultados/redacción, límites y evidencia. Biblioteca de instrucciones, motor de
orquestación, adapters de herramientas y estado persistido son piezas distintas:
los perfiles conservados no implementan por sí solos esas piezas de un producto.

Cada skill mantiene un snapshot exacto del perfil recibido para comparación,
sin usarlo como segunda capa de instrucciones de ejecución. Los snapshots no
declaran licencia de origen; no atribuirles la MIT de otras guías del catálogo.
El [registro de preservación](security-profiles-preservation.json) documenta
hashes, estructura conservada y adaptaciones para detectar recortes futuros.
