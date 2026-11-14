---
name: auth-implementation-patterns
description: "AI Toque: Master authentication and authorization patterns including JWT, OAuth2, session management, and RBAC to build secure, scalable access control systems. Use when implementing auth systems, securing APIs, or debugging security issues."
---

## Alcance y selección — AI Toque

Especialidad completa y reutilizable entre productos. Leer primero el
[contrato de ejecución](../../ai-toque-execution-context.md), que adapta
los supuestos del original y prevalece sobre sus recomendaciones genéricas,
y después el contexto del producto auditado o desarrollado.

Elegir esta skill por su resultado específico; leer el cuerpo completo
y las referencias pertinentes. Decidir aplicabilidad durante la ejecución,
sin borrar capacidades futuras ni activar otras guías automáticamente.
Las convenciones del producto, en especial `python-guidelines`, gobiernan
las decisiones locales. No instalar herramientas, imponer frameworks,
delegar ni ampliar permisos por cargar este perfil.

El [original recibido completo](../../originals/first-import/developer-essentials/skills/auth-implementation-patterns/original-skill.md) conserva los bytes y
metadata originales para trazabilidad. No es otra capa de instrucciones.
El [manifest de preservación](../../library-preservation.json) registra
integridad de cuerpos, referencias, templates y recursos.

### Recursos completos de la especialidad

- [references/details.md](references/details.md)

Seleccionar los recursos correspondientes al modo solicitado. Los templates
son ejemplos preservados; adaptarlos y validarlos antes de utilizarlos.

## Perfil especializado completo

# Authentication & Authorization Implementation Patterns

Build secure, scalable authentication and authorization systems using industry-standard patterns and modern best practices.

## When to Use This Skill

- Implementing user authentication systems
- Securing REST or GraphQL APIs
- Adding OAuth2/social login
- Implementing role-based access control (RBAC)
- Designing session management
- Migrating authentication systems
- Debugging auth issues
- Implementing SSO or multi-tenancy

## Core Concepts

### 1. Authentication vs Authorization

**Authentication (AuthN)**: Who are you?

- Verifying identity (username/password, OAuth, biometrics)
- Issuing credentials (sessions, tokens)
- Managing login/logout

**Authorization (AuthZ)**: What can you do?

- Permission checking
- Role-based access control (RBAC)
- Resource ownership validation
- Policy enforcement

### 2. Authentication Strategies

**Session-Based:**

- Server stores session state
- Session ID in cookie
- Traditional, simple, stateful

**Token-Based (JWT):**

- Stateless, self-contained
- Scales horizontally
- Can store claims

**OAuth2/OpenID Connect:**

- Delegate authentication
- Social login (Google, GitHub)
- Enterprise SSO

## Detailed patterns and worked examples

Detailed pattern documentation lives in `references/details.md`. Read that file when the navigation tier above is insufficient.

## Best Practices

1. **Never Store Plain Passwords**: Always hash with bcrypt/argon2
2. **Use HTTPS**: Encrypt data in transit
3. **Short-Lived Access Tokens**: 15-30 minutes max
4. **Secure Cookies**: httpOnly, secure, sameSite flags
5. **Validate All Input**: Email format, password strength
6. **Rate Limit Auth Endpoints**: Prevent brute force attacks
7. **Implement CSRF Protection**: For session-based auth
8. **Rotate Secrets Regularly**: JWT secrets, session secrets
9. **Log Security Events**: Login attempts, failed auth
10. **Use MFA When Possible**: Extra security layer

## Common Pitfalls

- **Weak Passwords**: Enforce strong password policies
- **JWT in localStorage**: Vulnerable to XSS, use httpOnly cookies
- **No Token Expiration**: Tokens should expire
- **Client-Side Auth Checks Only**: Always validate server-side
- **Insecure Password Reset**: Use secure tokens with expiration
- **No Rate Limiting**: Vulnerable to brute force
- **Trusting Client Data**: Always validate on server
