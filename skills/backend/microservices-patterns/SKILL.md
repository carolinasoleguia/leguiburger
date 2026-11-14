---
name: microservices-patterns
description: "AI Toque: Design microservices architectures with service boundaries, event-driven communication, and resilience patterns. Use when building distributed systems, decomposing monoliths, or implementing microservices."
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

El [original recibido completo](../../originals/second-import/backend-development/skills/microservices-patterns/original-skill.md) conserva los bytes y
metadata originales para trazabilidad. No es otra capa de instrucciones.
El [manifest de preservación](../../library-preservation.json) registra
integridad de cuerpos, referencias, templates y recursos.

### Recursos completos de la especialidad

- [references/details.md](references/details.md)

Seleccionar los recursos correspondientes al modo solicitado. Los templates
son ejemplos preservados; adaptarlos y validarlos antes de utilizarlos.

## Perfil especializado completo

# Microservices Patterns

Master microservices architecture patterns including service boundaries, inter-service communication, data management, and resilience patterns for building distributed systems.

## When to Use This Skill

- Decomposing monoliths into microservices
- Designing service boundaries and contracts
- Implementing inter-service communication
- Managing distributed data and transactions
- Building resilient distributed systems
- Implementing service discovery and load balancing
- Designing event-driven architectures

## Core Concepts

### 1. Service Decomposition Strategies

**By Business Capability**

- Organize services around business functions
- Each service owns its domain
- Example: OrderService, PaymentService, InventoryService

**By Subdomain (DDD)**

- Core domain, supporting subdomains
- Bounded contexts map to services
- Clear ownership and responsibility

**Strangler Fig Pattern**

- Gradually extract from monolith
- New functionality as microservices
- Proxy routes to old/new systems

### 2. Communication Patterns

**Synchronous (Request/Response)**

- REST APIs
- gRPC
- GraphQL

**Asynchronous (Events/Messages)**

- Event streaming (Kafka)
- Message queues (RabbitMQ, SQS)
- Pub/Sub patterns

### 3. Data Management

**Database Per Service**

- Each service owns its data
- No shared databases
- Loose coupling

**Saga Pattern**

- Distributed transactions
- Compensating actions
- Eventual consistency

### 4. Resilience Patterns

**Circuit Breaker**

- Fail fast on repeated errors
- Prevent cascade failures

**Retry with Backoff**

- Transient fault handling
- Exponential backoff

**Bulkhead**

- Isolate resources
- Limit impact of failures

## Detailed patterns and worked examples

Detailed pattern documentation lives in `references/details.md`. Read that file when the navigation tier above is insufficient.
