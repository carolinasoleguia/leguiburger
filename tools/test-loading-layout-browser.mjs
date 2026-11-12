import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile, mkdir, writeFile } from "node:fs/promises";
import { resolve, sep, extname } from "node:path";
import { pathToFileURL } from "node:url";
const { default: puppeteer } = await import(
  pathToFileURL(process.env.PUPPETEER_MODULE).href
);
const dist = resolve("frontend/dist");
const server = createServer(async (req, res) => {
  let file = resolve(dist, "." + new URL(req.url, "http://localhost").pathname);
  if (!file.startsWith(dist + sep) || !extname(file))
    file = resolve(dist, "index.html");
  try {
    const body = await readFile(file);
    res.setHeader(
      "Content-Type",
      {
        ".html": "text/html",
        ".js": "application/javascript",
        ".css": "text/css",
        ".woff2": "font/woff2",
        ".svg": "image/svg+xml",
      }[extname(file)] || "application/octet-stream",
    );
    res.end(body);
  } catch {
    res.writeHead(404);
    res.end();
  }
});
await new Promise((r) => server.listen(0, "127.0.0.1", r));
const origin = `http://127.0.0.1:${server.address().port}`;
const browser = await puppeteer.launch({
  headless: true,
  executablePath: process.env.CHROME_PATH,
  args: ["--disable-gpu", ...(process.env.CI ? ["--no-sandbox"] : [])],
});
await mkdir(".cache/loading-layout", { recursive: true });
const results = [],
  failures = [];
let page;
const scenarios = [
  "/locations",
  "/team",
  "/orders",
  "/cash",
  "/inventory",
  "/expenses",
  "/customers",
  "/kitchen",
  "/notifications",
  "/onboarding",
  "/restaurants",
  "/care",
  "/payments",
  "/account/security",
  "/team/invitations",
  "/audit",
  "/admin/catalog",
  "/admin/production",
  "/admin",
  "/owner",
  "/owner/brands",
  "cashier:/inventory",
];
function data(path, count) {
  const rows = Array.from({ length: count }, (_, i) => ({
    id: `fixture-${i}`,
    brand_id: "brand-1",
    tenant_id: "tenant-1",
    subdomain: `centro-${i}`,
    display_name: `Local ${i}`,
    name: `Doble cheddar ${i}`,
    first_name: "Ana",
    last_name: `Equipo ${i}`,
    email: `fixture-${i}@example.invalid`,
    role: "cashier",
    destination: "/inventory",
    is_active: true,
    active: true,
    location_ids: ["tenant-1"],
    created_at: "2026-11-11T12:00:00Z",
    updated_at: "2026-11-11T12:00:00Z",
    occurred_at: "2026-11-11T12:00:00Z",
    status: "pending",
    currency: "ARS",
    total_due: "12345.50",
    amount: "100.00",
    kind: "product",
    current_stock: "5",
    minimum_stock: "2",
    track_stock: true,
    unit: "unit",
    stock_version: 1,
    items: [],
    invitations: [],
    base_location_limit: 2,
    base_collaborator_limit: 5,
    maximum_locations: 2,
    maximum_collaborators: 5,
    locations: 1,
    collaborators: 1,
    primary_owner_id: "user-1",
    subject: `Consulta ${i}`,
    state: "open",
    body: `Mensaje de prueba ${i}`,
    platform: false,
    production_date: "2026-11-11",
    medallions_produced: 120,
    breads_purchased: 120,
    notes: "Jornada de prueba",
    image_url: "",
    base_price: 9000,
    is_available: true,
    order_number: i + 1,
    code: "fixture",
    event_name: "auth.password_verified",
    actor_name: "Ana",
  }));
  const brand = {
    id: "brand-1",
    name: "Hamburguesería de prueba",
    primary_owner_id: "user-1",
    invitations: [],
    base_location_limit: 2,
    base_collaborator_limit: 5,
    maximum_locations: 2,
    maximum_collaborators: 5,
    locations: 1,
    collaborators: 1,
  };
  if (path === "/api/tenants")
    return [
      {
        id: "tenant-1",
        brand_id: "brand-1",
        subdomain: "centro",
        name: "Centro",
        brand: {
          name: count
            ? "Hamburguesería del Barrio Centro - Parrilla y Hamburguesas"
            : "",
        },
        active: true,
      },
    ];
  if (path === "/api/business" || path === "/api/platform/restaurants")
    return { items: [brand] };
  if (path === "/api/business/locations" || path === "/api/business/team")
    return { items: rows };
  if (path === "/api/analytics/restaurant")
    return {
      metrics: { orders_created: count },
      money: [
        {
          currency: "ARS",
          sales: "12345.50",
          collected: "10000",
          operating_expenses: "100",
          estimated_operating_result: null,
          average_ticket: "1234.50",
          missing_cost_lines: 1,
        },
      ],
      previous_period: { sales: [{ currency: "ARS", amount: "10000" }] },
      daily_sales: count
        ? [{ day: "2026-11-11", currency: "ARS", total: 12345.5 }]
        : [],
      top_products: rows.slice(0, 5).map((x) => ({
        ...x,
        product_id: x.id,
        product_name: x.name,
        units: 3,
      })),
      recent_orders: rows,
    };
  if (path === "/api/analytics/platform")
    return {
      businesses: {
        production_registered_in_period: count,
        active_last_week: count,
        unclassified: count,
      },
      subscriptions: { active: count },
      money: [{ currency: "ARS", collected: "12345.50", active_mrr: "15000" }],
      daily_new_businesses: count
        ? [{ day: "2026-11-11", registered: count }]
        : [],
      activity_by_feature: rows.map((x) => ({
        ...x,
        events: count,
        businesses: count,
      })),
    };
  if (
    path === "/api/pos/catalog" ||
    path === "/api/orders" ||
    path === "/api/crm/customers" ||
    path === "/api/expenses" ||
    path === "/api/inventory"
  )
    return { items: rows };
  if (path === "/api/cash/shifts")
    return {
      items: rows.map((x, i) => ({
        ...x,
        opened_at: "2026-11-12T10:00:00Z",
        closed_at: i ? "2026-11-12T11:00:00Z" : null,
        opening_balance: "100.00",
        expected_balance: i ? "100.00" : null,
        counted_balance: i ? "100.00" : null,
        difference: i ? "0.00" : null,
      })),
    };
  if (/^\/api\/cash\/shifts\/[^/]+$/.test(path))
    return {
      id: "fixture-0",
      currency: "ARS",
      opened_at: "2026-11-12T10:00:00Z",
      closed_at: null,
      opening_balance: "100.00",
      expected_balance: "120.00",
      cash_charged: "20.00",
      cash_refunded: "0.00",
    };
  if (path.endsWith("/entries") && path.startsWith("/api/cash/"))
    return {
      items: rows.map((x) => ({
        ...x,
        kind: "cash_in",
        amount: "1.00",
        note: "Cambio de turno",
      })),
    };
  if (path === "/api/kitchen")
    return { items: rows.map((x) => ({ ...x, status: "accepted" })) };
  if (path === "/api/notifications")
    return {
      items: rows.map((x) => ({
        ...x,
        title: x.subject,
        message: x.body,
        destination: "/inventory",
        read_at: null,
      })),
      unread: count,
    };
  if (path === "/api/onboarding")
    return {
      completed: 0,
      total: 4,
      steps: [
        {
          key: "store",
          title: "Configurá tu local",
          description: "Datos de contacto y horarios",
          done: !!count,
          href: "/locations",
          to: "/locations",
        },
      ],
    };
  if (
    path === "/api/products" ||
    path === "/api/production" ||
    path === "/api/brands"
  )
    return rows;
  if (path === "/api/auth/sessions")
    return rows.map((x) => ({
      ...x,
      current: false,
      last_seen_at: x.created_at,
      user_agent: "Dispositivo de prueba",
    }));
  if (path === "/api/team/invitations")
    return rows.map((x) => ({ ...x, expires_at: "2026-12-11T00:00:00Z" }));
  if (path === "/api/audit") return { items: rows };
  if (path === "/api/payments/mercadopago/config")
    return { enabled: false, environment: "sandbox" };
  if (path === "/api/billing/my-account")
    return { invoices: [], subscriptions: [] };
  if (path === "/api/billing/plans") return [];
  if (
    path === "/api/payments/mercadopago/transactions" ||
    path === "/api/payments/mercadopago/checkouts" ||
    path === "/api/payments/mercadopago/reviews"
  )
    return { items: [] };
  if (path === "/api/integrations/mercadopago/connections") return [];
  if (path === "/api/care/support") return rows;
  if (path === "/api/care/backups" || path === "/api/care/errors") return [];
  if (path === "/api/care/closure")
    return { name: brand.name, primary: true, closed_at: null, renewals: [] };
  return { items: [] };
}
const selectors = [
  "main .section-header",
  "main>.operations-page",
  "main .workspace-filters",
  "main .stable-content",
  "main .stable-content-body>.table-scroll",
  "main .stable-content-body>form",
  "main .metric-grid",
  "main .report-grid",
  "main .cash-summary",
  "main .cash-forms",
  "main .catalog-tabs",
  "main .tenant-picker-card",
  "main .stable-table-region",
  "main .payment-tabs",
  "main .payment-filters",
  "main .care-grid",
  ...Array.from({ length: 8 }, (_, i) => `main table th:nth-child(${i + 1})`),
];
async function boxes() {
  return page.evaluate(
    (selectors) =>
      Object.fromEntries(
        selectors.map((selector) => {
          const e = document.querySelector(selector);
          if (!e) return [selector, null];
          const r = e.getBoundingClientRect();
          return [
            selector,
            { x: r.x, y: r.y, width: r.width, height: r.height },
          ];
        }),
      ),
    selectors,
  );
}
try {
  for (const width of [1440, 390])
    for (const scenario of scenarios)
      for (const count of [0, 25]) {
        const [overrideRole, path] = scenario.includes(":")
          ? scenario.split(":")
          : [null, scenario];
        const role =
            overrideRole || (path.startsWith("/owner") ? "owner" : "admin"),
          queued = [],
          errors = [];
        let holding = true;
        page = await browser.newPage();
        page.on("pageerror", (e) => errors.push(e.message));
        await page.setViewport({ width, height: 1000 });
        await page.emulateMediaFeatures([
          { name: "prefers-reduced-motion", value: "no-preference" },
        ]);
        await page.setRequestInterception(true);
        page.on("request", (r) => {
          const p = new URL(r.url()).pathname;
          if (!p.startsWith("/api/")) return r.continue();
          if (p === "/api/auth/refresh")
            return r.respond({
              status: 200,
              contentType: "application/json",
              body: JSON.stringify({
                token: "fixture",
                user: {
                  id: "user-1",
                  role,
                  brand_id: "brand-1",
                  email: "fixture@example.invalid",
                  tenant_id: role === "cashier" ? "tenant-1" : undefined,
                },
              }),
            });
          if (holding) {
            queued.push(r);
            return;
          }
          return r.respond({
            status: 200,
            contentType: "application/json",
            body: JSON.stringify(data(p, count)),
          });
        });
        await page.goto(origin + path, { waitUntil: "domcontentloaded" });
        await page.waitForSelector("[data-dashboard-loading]");
        await page.evaluate(async () => {
          await document.fonts.ready;
          await Promise.allSettled(
            document
              .getAnimations()
              .filter(
                (a) => a.effect?.getComputedTiming().iterations !== Infinity,
              )
              .map((a) => a.finished),
          );
          window.__shifts = [];
          window.__fade = false;
          document.addEventListener("animationstart", (e) => {
            if (e.animationName === "stable-fade-in") window.__fade = true;
          });
          window.__observer = new PerformanceObserver((list) => {
            for (const entry of list.getEntries())
              if (!entry.hadRecentInput)
                window.__shifts.push({
                  value: entry.value,
                  nodes: entry.sources.map(
                    (s) => s.node?.className || s.node?.nodeName,
                  ),
                });
          });
          window.__observer.observe({ type: "layout-shift" });
        });
        const before = await boxes();
        holding = false;
        await Promise.all(
          queued.splice(0).map((r) =>
            r.respond({
              status: 200,
              contentType: "application/json",
              body: JSON.stringify(data(new URL(r.url()).pathname, count)),
            }),
          ),
        );
        await page.waitForFunction(
          () => !document.querySelector("[data-dashboard-loading]"),
        );
        await page.evaluate(async () => {
          await Promise.allSettled(
            document
              .getAnimations()
              .filter(
                (a) => a.effect?.getComputedTiming().iterations !== Infinity,
              )
              .map((a) => a.finished),
          );
        });
        if (await page.$(".recovery-page"))
          throw new Error(`Fixture failed to render ${path}`);
        const after = await boxes(),
          metrics = await page.evaluate(() => ({
            cls: window.__shifts.reduce((s, e) => s + e.value, 0),
            shifts: window.__shifts,
            fade: window.__fade,
            overflow: document.documentElement.scrollWidth > innerWidth + 1,
          }));
        const drift = [];
        for (const selector of selectors) {
          const a = before[selector],
            b = after[selector];
          if (!a || !b) continue;
          for (const property of ["x", "y", "width"])
            if (Math.abs(a[property] - b[property]) > 1)
              drift.push({
                selector,
                property,
                before: a[property],
                after: b[property],
              });
          // Support adds account-specific sections below the fold; compare its
          // visible panel anchors and CLS rather than the whole document length.
          if (
            (selector.includes("stable-content") && path !== "/care") ||
            selector.includes("metric-grid") ||
            selector.includes("report-grid") ||
            selector.includes("stable-table-region")
          )
            if (Math.abs(a.height - b.height) > 1)
              drift.push({
                selector,
                property: "height",
                before: a.height,
                after: b.height,
              });
        }
        results.push({ path, role, width, count, ...metrics, drift });
        if (
          errors.length ||
          drift.length ||
          metrics.cls > 0.001 ||
          !metrics.fade ||
          metrics.overflow
        ) {
          await page.screenshot({
            path: `.cache/loading-layout/failure-${path.split("/").filter(Boolean).join("-")}-${width}-${count}.png`,
            fullPage: true,
          });
          console.error(JSON.stringify(results.at(-1), null, 2), errors);
          failures.push({ path, width, count, ...metrics, drift, errors });
        }
        await page.close();
      }
  await writeFile(
    ".cache/loading-layout/results.json",
    JSON.stringify(results, null, 2),
  );
  if (failures.length)
    throw new Error(
      `${failures.length} loading layouts need adjustment; see .cache/loading-layout/results.json`,
    );
  console.log(
    `Loading layout passed: ${results.length} empty/populated desktop/mobile scenarios; reserved frames, stable anchors, fade and CLS checked.`,
  );
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
  await writeFile(
    ".cache/loading-layout/results.json",
    JSON.stringify(results, null, 2),
  );
}
