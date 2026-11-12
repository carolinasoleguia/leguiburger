---
name: e2e-testing-patterns
description: "AI Toque: Master end-to-end testing with Playwright and Cypress to build reliable test suites that catch bugs, improve confidence, and enable fast deployment. Use when implementing E2E tests, debugging flaky tests, or establishing testing standards."
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

El [original recibido completo](../../originals/first-import/developer-essentials/skills/e2e-testing-patterns/original-skill.md) conserva los bytes y
metadata originales para trazabilidad. No es otra capa de instrucciones.
El [manifest de preservación](../../library-preservation.json) registra
integridad de cuerpos, referencias, templates y recursos.

### Recursos completos de la especialidad

- [references/details.md](references/details.md)

Seleccionar los recursos correspondientes al modo solicitado. Los templates
son ejemplos preservados; adaptarlos y validarlos antes de utilizarlos.

## Perfil especializado completo

# E2E Testing Patterns

Build reliable, fast, and maintainable end-to-end test suites that provide confidence to ship code quickly and catch regressions before users do.

## When to Use This Skill

- Implementing end-to-end test automation
- Debugging flaky or unreliable tests
- Testing critical user workflows
- Setting up CI/CD test pipelines
- Testing across multiple browsers
- Validating accessibility requirements
- Testing responsive designs
- Establishing E2E testing standards

## Core Concepts

### 1. E2E Testing Fundamentals

**What to Test with E2E:**

- Critical user journeys (login, checkout, signup)
- Complex interactions (drag-and-drop, multi-step forms)
- Cross-browser compatibility
- Real API integration
- Authentication flows

**What NOT to Test with E2E:**

- Unit-level logic (use unit tests)
- API contracts (use integration tests)
- Edge cases (too slow)
- Internal implementation details

### 2. Test Philosophy

**The Testing Pyramid:**

```
        /\
       /E2E\         ← Few, focused on critical paths
      /─────\
     /Integr\        ← More, test component interactions
    /────────\
   /Unit Tests\      ← Many, fast, isolated
  /────────────\
```

**Best Practices:**

- Test user behavior, not implementation
- Keep tests independent
- Make tests deterministic
- Optimize for speed
- Use data-testid, not CSS selectors

## Detailed patterns and worked examples

Detailed pattern documentation lives in `references/details.md`. Read that file when the navigation tier above is insufficient.

## Best Practices

1. **Use Data Attributes**: `data-testid` or `data-cy` for stable selectors
2. **Avoid Brittle Selectors**: Don't rely on CSS classes or DOM structure
3. **Test User Behavior**: Click, type, see - not implementation details
4. **Keep Tests Independent**: Each test should run in isolation
5. **Clean Up Test Data**: Create and destroy test data in each test
6. **Use Page Objects**: Encapsulate page logic
7. **Meaningful Assertions**: Check actual user-visible behavior
8. **Optimize for Speed**: Mock when possible, parallel execution

```typescript
// ❌ Bad selectors
cy.get(".btn.btn-primary.submit-button").click();
cy.get("div > form > div:nth-child(2) > input").type("text");

// ✅ Good selectors
cy.getByRole("button", { name: "Submit" }).click();
cy.getByLabel("Email address").type("user@example.com");
cy.get('[data-testid="email-input"]').type("user@example.com");
```

## Common Pitfalls

- **Flaky Tests**: Use proper waits, not fixed timeouts
- **Slow Tests**: Mock external APIs, use parallel execution
- **Over-Testing**: Don't test every edge case with E2E
- **Coupled Tests**: Tests should not depend on each other
- **Poor Selectors**: Avoid CSS classes and nth-child
- **No Cleanup**: Clean up test data after each test
- **Testing Implementation**: Test user behavior, not internals

## Debugging Failing Tests

```typescript
// Playwright debugging
// 1. Run in headed mode
npx playwright test --headed

// 2. Run in debug mode
npx playwright test --debug

// 3. Use trace viewer
await page.screenshot({ path: 'screenshot.png' });
await page.video()?.saveAs('video.webm');

// 4. Add test.step for better reporting
test('checkout flow', async ({ page }) => {
    await test.step('Add item to cart', async () => {
        await page.goto('/products');
        await page.getByRole('button', { name: 'Add to Cart' }).click();
    });

    await test.step('Proceed to checkout', async () => {
        await page.goto('/cart');
        await page.getByRole('button', { name: 'Checkout' }).click();
    });
});

// 5. Inspect page state
await page.pause();  // Pauses execution, opens inspector
```
