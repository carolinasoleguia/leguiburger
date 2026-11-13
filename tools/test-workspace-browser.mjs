import { paidMembership } from "./membership-fixture.mjs";
import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile, mkdir } from "node:fs/promises";
import { resolve, sep, extname } from "node:path";
import { pathToFileURL } from "node:url";
if (!process.env.PUPPETEER_MODULE)
  throw new Error("Set PUPPETEER_MODULE and CHROME_PATH");
const { default: puppeteer } = await import(
  pathToFileURL(process.env.PUPPETEER_MODULE).href
);
const dist = resolve("frontend/dist");
const server = createServer(async (request, response) => {
  let file = resolve(
    dist,
    "." + new URL(request.url, "http://localhost").pathname,
  );
  if (!file.startsWith(dist + sep) || !extname(file))
    file = resolve(dist, "index.html");
  try {
    const content = await readFile(file);
    response.setHeader(
      "Content-Type",
      {
        ".html": "text/html",
        ".js": "application/javascript",
        ".css": "text/css",
        ".woff2": "font/woff2",
        ".svg": "image/svg+xml",
      }[extname(file)] || "application/octet-stream",
    );
    response.end(content);
  } catch {
    response.writeHead(404);
    response.end();
  }
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
const browser = await puppeteer.launch({
  headless: true,
  executablePath: process.env.CHROME_PATH,
  args: ["--disable-gpu", ...(process.env.CI ? ["--no-sandbox"] : [])],
});
await mkdir(".cache/workspace-browser", { recursive: true });
let role = "admin",
  stock = "5",
  version = 1,
  failMovement = true,
  read = false;
const captured = [],
  errors = [];
let page;
const person = () => ({
  token: "fixture-token",
  user: {
    id: "workspace-fixture",
    role,
    brand_id: "brand-1",
    tenant_id: ["employee", "kitchen", "cashier"].includes(role)
      ? "tenant-1"
      : undefined,
    email: "fixture@example.invalid",
  },
});
const item = () => ({
  id: "product-1",
  kind: "product",
  name: "Doble cheddar",
  current_stock: stock,
  minimum_stock: "2",
  stock_version: version,
  track_stock: true,
  unit: "unit",
});
const report = {
  metrics: { orders_created: 4 },
  money: [
    {
      currency: "ARS",
      sales: "10000",
      average_ticket: "2500",
      collected: "9000",
      operating_expenses: "1000",
      estimated_operating_result: null,
      missing_cost_lines: 1,
    },
  ],
  daily_sales: [{ day: "2026-11-10", currency: "ARS", total: 10000 }],
  top_products: [
    {
      product_id: "product-1",
      product_name: "Doble cheddar",
      units: 4,
      currency: "ARS",
    },
  ],
  recent_orders: [],
  previous_period: { sales: [{ currency: "ARS", amount: 5000 }] },
};
try {
  page = await browser.newPage();
  page.on("dialog", (dialog) => dialog.accept());
  page.on("pageerror", (e) => errors.push(e.message));
  await page.setRequestInterception(true);
  page.on("request", (request) => {
    if (new URL(request.url()).pathname === '/api/membership') return request.respond({ status: 200, contentType: 'application/json', body: JSON.stringify(paidMembership) });
    const u = new URL(request.url()),
      p = u.pathname;
    if (!p.startsWith("/api/")) return request.continue();
    const body = JSON.parse(request.postData() || "{}");
    let result = {},
      status = 200;
    if (p === "/api/auth/refresh" || p === "/api/auth/login") result = person();
    else if (p === "/api/tenants")
      result = [
        {
          id: "tenant-1",
          brand_id: "brand-1",
          subdomain: "Centro",
          active: true,
        },
      ];
    else if (p === "/api/analytics/restaurant") result = report;
    else if (p === "/api/analytics/platform")
      result = {
        businesses: {
          production_registered_in_period: 2,
          active_last_week: 2,
          unclassified: 1,
        },
        subscriptions: { active: 1 },
        money: [{ currency: "ARS", collected: 5000, active_mrr: 5000 }],
        daily_new_businesses: [],
        activity_by_feature: [],
      };
    else if (p === "/api/inventory") result = { items: [item()] };
    else if (p === "/api/inventory/movements" && request.method() === "POST") {
      captured.push({
        type: "movement",
        body,
        key: request.headers()["idempotency-key"],
      });
      if (failMovement) {
        status = 503;
        result = {
          message: "Respuesta incierta. Reintentá la misma operación.",
        };
        failMovement = false;
      } else {
        stock = "7";
        version = 2;
        result = { id: "movement-1" };
      }
    } else if (p === "/api/inventory/movements")
      result = {
        items: [
          {
            id: "movement-1",
            occurred_at: new Date().toISOString(),
            reason: "purchase",
            item_kind: "product",
            quantity_delta: "2",
            balance_before: "5",
            balance_after: "7",
          },
        ],
      };
    else if (p === "/api/expenses" && request.method() === "POST") {
      captured.push({ type: "expense", body });
      result = { id: "expense-1" };
    } else if (p === "/api/expenses") result = { items: [] };
    else if (p === "/api/kitchen/load")
      result = {
        active_count: 1,
        max_active_orders: 0,
        full: false,
        prep_min: 20,
        prep_max: 35,
      };
    else if (p === "/api/kitchen")
      result = {
        items: [
          {
            id: "order-12345678",
            status: "accepted",
            version: 1,
            created_at: new Date().toISOString(),
            items: [
              {
                product_name: "Doble cheddar",
                quantity: 2,
                extras: [{ name: "Bacon", quantity: 1 }],
              },
            ],
          },
        ],
      };
    else if (p.startsWith("/api/kitchen/") && request.method() === "PATCH") {
      captured.push({ type: "kitchen", body });
      result = { status: "preparing", version: 2 };
    } else if (p === "/api/pos/catalog")
      result = {
        items: [{ id: "product-1", name: "Doble cheddar", price: "2500.00" }],
      };
    else if (p === "/api/orders" && request.method() === "POST") {
      captured.push({ type: "order", body });
      result = { id: "order-12345678" };
    } else if (p === "/api/orders") result = { items: [] };
    else if (p === "/api/orders/inbox")
      result = { items: [], pending_count: 0 };
    else if (p === "/api/orders/events") {
      status = 503;
      result = {};
    } else if (p === "/api/crm/customers")
      result = {
        items: [
          {
            id: "customer-1",
            first_name: "Ana",
            last_name: "Cliente",
            email: "ana@example.invalid",
            orders: 2,
          },
        ],
      };
    else if (p === "/api/customers" && request.method() === "POST") {
      captured.push({ type: "customer", body });
      result = { ID: "customer-2" };
    } else if (p === "/api/notifications")
      result = {
        items: [
          {
            id: "notification-1",
            title: "Cobro confirmado",
            message: "Se acreditó el cobro.",
            destination: "/orders",
            created_at: new Date().toISOString(),
            read_at: read ? new Date().toISOString() : null,
          },
        ],
        unread: read ? 0 : 1,
      };
    else if (p === "/api/notifications/read") {
      read = true;
      result = { ok: true };
    } else if (p === "/api/onboarding")
      result = {
        steps: ["brand", "location", "menu", "team"].map((key) => ({
          key,
          title: key,
          done: true,
          to: "/admin",
        })),
      };
    else if (p === "/api/onboarding/complete") {
      captured.push({ type: "onboarding" });
      result = { completed_at: new Date().toISOString() };
    } else if (p === "/api/platform/businesses") result = { items: [] };
    else if (p === "/api/platform/health")
      result = {
        workers: [
          { instance_id: "worker-1", last_seen_at: new Date().toISOString() },
        ],
        mail_pending: 0,
        mail_failed: 0,
        webhooks_pending: 0,
      };
    else if (p === "/api/public/order") {
      assert.equal(
        request.headers().authorization,
        "Bearer fixture-capability",
      );
      result = {
        business: "LeguiBurger",
        status: "preparing",
        currency: "ARS",
        total_due: 5000,
        total_paid: 5000,
        items: [{ name: "Doble cheddar", quantity: 2 }],
      };
    } else if (p === "/api/auth/email/confirm") result = { completed: true };
    else if (p === "/api/auth/sessions") result = [];
    else if (p === "/api/auth/email/change") {
      captured.push({ type: "email", body });
      result = { message: "Enviamos una confirmación a ambos correos." };
    } else {
      status = 404;
      result = { message: "Unexpected fixture route " + p };
    }
    request.respond({
      status,
      contentType: "application/json",
      body: JSON.stringify(result),
    });
  });
  const goto = async (path) => {
    await page.goto(origin + path, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.querySelector("#app")?.childElementCount > 0,
    );
    await page.waitForNetworkIdle({ idleTime: 100, timeout: 10000 });
  };
  const click = async (text) =>
    page.evaluate((text) => {
      const b = [...document.querySelectorAll("button")].find(
        (x) => x.textContent.trim() === text,
      );
      if (!b) throw new Error("Missing " + text);
      b.click();
    }, text);
  const fill = async (label, value) =>
    page.evaluate(
      ({ label, value }) => {
        const labels = [...document.querySelectorAll("label")];
        const el = (
          labels.find((x) => x.firstChild?.textContent.trim() === label) ||
          labels.find((x) => x.textContent.trim().startsWith(label))
        )?.querySelector("input,select,textarea");
        if (!el) throw new Error("Missing field " + label);
        el.value = value;
        el.dispatchEvent(
          new Event(el.tagName === "SELECT" ? "change" : "input", {
            bubbles: true,
          }),
        );
      },
      { label, value },
    );
  await goto("/admin");
  assert(await page.$eval("main", (x) => x.textContent.includes("Sin datos")));
  assert(await page.$eval("main", (x) => x.textContent.includes("100.0%")));
  await goto("/inventory");
  await click("Gestionar");
  await fill("Cantidad a sumar", "2");
  await click("Registrar movimiento");
  await page.waitForSelector('[role="alert"]');
  await click("Registrar movimiento");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Movimiento registrado"),
  );
  assert.equal(captured[0].key, captured[1].key);
  assert.equal(captured[1].body.quantity_delta, "2");
  assert.equal(captured[1].body.expected_version, 1);
  await click("Historial");
  await page.waitForFunction(() =>
    document.querySelector("main").textContent.includes("5 → 7"),
  );
  await goto("/expenses");
  await fill("Importe en pesos", "120.50");
  await click("Guardar gasto");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Gasto registrado"),
  );
  assert.equal(
    captured.find((x) => x.type === "expense").body.amount,
    "120.50",
  );
  role = "cashier";
  await goto("/orders");
  await fill("Hamburguesa", "product-1");
  await click("Sumar al pedido");
  await click("Crear pedido");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Pedido creado"),
  );
  assert.equal(
    captured.find((x) => x.type === "order").body.items[0].quantity,
    1,
  );
  await goto("/kitchen");
  await click("Empezar preparación");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Preparación actualizada"),
  );
  assert.equal(
    captured.find((x) => x.type === "kitchen").body.expected_version,
    1,
  );
  await goto("/customers");
  await fill("Nombre", "Nuevo");
  await fill("Apellido", "Cliente");
  await fill("Correo", "nuevo@example.invalid");
  await click("Guardar cliente");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Cliente guardado"),
  );
  await goto("/notifications");
  await click("Marcar como leída");
  await page.waitForFunction(() =>
    document.querySelector("main").textContent.includes("0 sin leer"),
  );
  role = "admin";
  await goto("/onboarding");
  await click("Finalizar preparación");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("listo para trabajar"),
  );
  for (const width of [1440, 390]) {
    await page.setViewport({ width, height: 960 });
    for (const route of [
      "/admin",
      "/inventory",
      "/expenses",
      "/orders",
      "/kitchen",
      "/customers",
      "/notifications",
      "/onboarding",
    ]) {
      role = ["/orders", "/kitchen"].includes(route) ? "cashier" : "admin";
      await goto(route);
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
        `${route} overflows at ${width}`,
      );
      await page.screenshot({
        path: `.cache/workspace-browser/${route.slice(1)}-${width}.png`,
        fullPage: true,
      });
    }
  }
  role = "owner";
  await goto("/owner");
  await goto("/platform/health");
  assert(
    await page.$eval("main", (x) => x.textContent.includes("Worker activo")),
  );
  role = "kitchen";
  await goto("/tenant/tenant-1/employee");
  assert(
    await page.$eval("main", (x) => x.textContent.includes("Doble cheddar")),
  );
  await goto("/expenses");
  assert.equal(new URL(page.url()).pathname, "/unauthorized");
  await goto("/order#token=fixture-capability");
  assert.equal(new URL(page.url()).hash, "");
  assert(
    await page.$eval("main", (x) => x.textContent.includes("En preparación")),
  );
  await goto("/confirm-email#token=fixture-email");
  assert.equal(new URL(page.url()).hash, "");
  await click("Confirmar esta casilla");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Correo actualizado"),
  );
  assert.deepEqual(errors, []);
  console.log(
    "Workspace browser passed: real dashboards/null margins, exact decimals, retry idempotency, stock history, expenses, POS, kitchen, CRM, notifications, onboarding, roles, private tracking, email confirmation and responsive layouts at 1440/390.",
  );
} catch (e) {
  console.error("Browser errors:", errors, "URL:", page?.url());
  console.error(await page?.evaluate(() => document.body.innerText));
  await page?.screenshot({
    path: ".cache/workspace-browser/failure.png",
    fullPage: true,
  });
  throw e;
} finally {
  await browser.close();
  await new Promise((resolve) => server.close(resolve));
}
