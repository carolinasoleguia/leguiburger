import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile, mkdir } from "node:fs/promises";
import { resolve, extname, sep } from "node:path";
import { pathToFileURL } from "node:url";
const { default: puppeteer } = await import(
  pathToFileURL(process.env.PUPPETEER_MODULE).href
);
const dist = resolve("frontend/dist");
let role = "owner",
  expired = false,
  premium = false,
  failDraft = true;
const calls = [],
  errors = [],
  prices = [];
const access = () => ({
  tier: expired ? "trial" : premium ? "premium" : "trial",
  state: expired ? "expired" : premium ? "paid" : "trial",
  operational: !expired,
  premium,
  locations: premium ? 3 : 1,
  terminals_per_location: premium ? 5 : 2,
  valid_until: new Date(
    Date.now() + (expired ? -1 : 7) * 86400000,
  ).toISOString(),
});
const publicPrices = [
  {
    id: "premium-price",
    name: "Premium",
    amount: "1000",
    currency: "ARS",
    interval_months: 1,
    billing_unit: "brand",
    membership_tier: "premium",
    extra_location_amount: "250",
  },
  {
    id: "classic-price",
    name: "Clásico",
    amount: "600",
    currency: "ARS",
    interval_months: 1,
    billing_unit: "brand",
    membership_tier: "standard",
    extra_location_amount: "0",
  },
];
const server = createServer(async (req, res) => {
  const path = new URL(req.url, "http://localhost").pathname;
  const reply = (body, status = 200) => {
    res.writeHead(status, { "Content-Type": "application/json" });
    res.end(JSON.stringify(body));
  };
  if (path.startsWith("/api/")) {
    let raw = "";
    for await (const chunk of req) raw += chunk;
    const body = JSON.parse(raw || "{}");
    calls.push({
      path,
      method: req.method,
      body,
      key: req.headers["idempotency-key"],
    });
    if (path === "/api/auth/refresh")
      return reply({
        token: "fixture",
        user: {
          id: "fixture-" + role,
          role,
          brand_id: "brand",
          email: "fixture@example.invalid",
          tenant_id: role === "cashier" ? "tenant" : undefined,
        },
      });
    if (path === "/api/membership") return reply(access());
    if (path === "/api/business") return reply({ items: [{ id: "brand" }] });
    if (path === "/api/tenants")
      return reply([
        { id: "tenant", brand_id: "brand", subdomain: "Centro", active: true },
      ]);
    if (path === "/api/brands") return reply([{ id: "brand", name: "Burger" }]);
    if (path === "/api/business/locations")
      return reply({
        items: [{ id: "tenant", tenant_id: "tenant", subdomain: "Centro" }],
      });
    if (path === "/api/payments/mercadopago/config")
      return reply({
        enabled: true,
        environment: "sandbox",
        platform_configured: true,
        oauth_configured: true,
      });
    if (path === "/api/platform/billing/plans")
      return reply(
        req.method === "POST"
          ? { id: "plan" }
          : { items: [{ id: "plan", name: "Premium", prices }] },
      );
    if (path === "/api/platform/billing/plans/plan/prices") {
      if (failDraft) {
        failDraft = false;
        return reply({ message: "Conexión interrumpida; reintentá." }, 503);
      }
      const price = {
        ...body,
        id: "price-" + (prices.length + 1),
        version: prices.length + 1,
        is_active: false,
      };
      prices.push(price);
      return reply(price, 201);
    }
    if (path.includes("/publication")) {
      const p = prices.find((p) => path.includes(p.id));
      p.is_active = body.published;
      return reply(p);
    }
    if (path === "/api/billing/plans") return reply({ items: publicPrices });
    if (path === "/api/billing/terminals") return reply({ items: [] });
    if (path === "/api/billing/my-account")
      return reply({ subscriptions: [], invoices: [] });
    if (path === "/api/billing/subscribe")
      return reply({ subscription_id: "subscription" }, 201);
    if (path === "/api/billing/subscriptions/subscription/checkout")
      return reply({
        checkout_url:
          "https://www.mercadopago.com.ar/subscriptions/checkout?fixture=membership",
      });
    return reply({ items: [] });
  }
  let file = resolve(dist, "." + path);
  if (!file.startsWith(dist + sep) || !extname(file))
    file = resolve(dist, "index.html");
  try {
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
    res.end(await readFile(file));
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
  args: process.env.CI ? ["--no-sandbox"] : [],
});
await mkdir(".cache/membership-browser", { recursive: true });
let page;
async function click(text) {
  await page.waitForFunction(
    (t) =>
      [...document.querySelectorAll("button,a")].some(
        (b) => b.textContent.trim() === t && !b.disabled,
      ),
    {},
    text,
  );
  await page.evaluate(
    (t) =>
      [...document.querySelectorAll("button,a")]
        .find((b) => b.textContent.trim() === t && !b.disabled)
        .click(),
    text,
  );
}
async function fill(selector, value) {
  await page.$eval(
    selector,
    (e, v) => {
      e.value = v;
      e.dispatchEvent(new Event("input", { bubbles: true }));
    },
    value,
  );
}
try {
  page = await browser.newPage();
  page.on("pageerror", (e) => errors.push(e.message));
  await page.setViewport({ width: 1366, height: 768 });
  await page.goto(origin + "/platform/plans", { waitUntil: "networkidle0" });
  await page.waitForSelector("#new-plan-name");
  await fill("#new-plan-name", "Premium");
  await click("Crear plan");
  await page.waitForFunction(
    () => document.querySelector("#pricing-plan")?.value === "plan",
  );
  await page.select("#membership-tier", "premium");
  await fill("#new-plan-price", "0");
  await click("Guardar borrador");
  await page.waitForFunction(() =>
    document.body.textContent.includes("Conexión interrumpida"),
  );
  await click("Guardar borrador");
  await page.waitForFunction(() =>
    document.body.textContent.includes("Tarifa guardada en borrador"),
  );
  const attempts = calls.filter((c) => c.path.endsWith("/prices"));
  assert.equal(attempts.length, 2);
  assert.equal(attempts[0].key, attempts[1].key);
  assert.equal(attempts[1].body.published, false);
  assert.equal(attempts[1].body.membership_tier, "premium");
  assert.equal(
    await page.$eval(".data-table tbody tr button", (e) => e.disabled),
    true,
  );
  await fill("#new-plan-price", "1000");
  await fill("#extra-location-price", "250");
  await click("Guardar borrador");
  await page.waitForFunction(
    () => document.querySelectorAll(".data-table tbody tr").length >= 2,
  );
  await click("Publicar");
  await page.waitForFunction(() =>
    document.body.textContent.includes(
      "Tarifa publicada para nuevas contrataciones",
    ),
  );
  assert.equal(
    calls.filter((c) => c.path.includes("/publication")).at(-1).body.published,
    true,
  );
  await page.waitForFunction(() =>
    [...document.querySelectorAll(".stable-content")].every(
      (e) => e.getAttribute("aria-busy") === "false",
    ),
  );
  await page.waitForFunction(() =>
    document
      .getAnimations()
      .every(
        (a) =>
          a.effect?.getTiming().iterations === Infinity ||
          ["finished", "idle"].includes(a.playState),
      ),
  );
  for (const width of [1366, 390]) {
    await page.setViewport({ width, height: 844 });
    for (const theme of ["light", "dark"]) {
      await page.evaluate(
        (theme) => document.documentElement.setAttribute("data-theme", theme),
        theme,
      );
      await page.waitForFunction(() =>
        document
          .getAnimations()
          .every(
            (a) =>
              a.effect?.getTiming().iterations === Infinity ||
              ["finished", "idle"].includes(a.playState),
          ),
      );
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      );
      await page.screenshot({
        path: `.cache/membership-browser/plans-${width}-${theme}.png`,
        fullPage: true,
      });
    }
  }
  role = "admin";
  expired = true;
  await page.goto(origin + "/admin", { waitUntil: "networkidle0" });
  assert.equal(new URL(page.url()).pathname, "/admin/membership");
  assert(
    (await page.$eval("main", (e) => e.textContent)).includes(
      "Renová para seguir operando",
    ),
  );
  await click("Ver planes y renovar");
  await page.waitForSelector("#membership-plan");
  await page.select("#membership-plan", "premium-price");
  await fill("#membership-locations", "3");
  await page.waitForFunction(() =>
    document.querySelector(".membership-total")?.textContent.includes("1.500"),
  );
  await click("Continuar con Mercado Pago");
  await page.waitForSelector("#checkout-link");
  assert.equal(
    calls.find((c) => c.path === "/api/billing/subscribe").body
      .purchased_locations,
    3,
  );
  for (const width of [1366, 390]) {
    await page.setViewport({ width, height: 844 });
    assert(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    );
    await page.screenshot({
      path: `.cache/membership-browser/renewal-${width}.png`,
      fullPage: true,
    });
  }
  role = "cashier";
  await page.goto(origin + "/orders", { waitUntil: "networkidle0" });
  assert.equal(new URL(page.url()).pathname, "/employee/membership");
  assert(
    !(await page.$eval("main", (e) => e.textContent)).includes(
      "Ver planes y renovar",
    ),
  );
  expired = false;
  await page.goto(origin + "/inventory", { waitUntil: "networkidle0" });
  assert.equal(new URL(page.url()).pathname, "/employee/membership");
  assert.deepEqual(errors, []);
  console.log(
    "Membership browser passed: draft retry/publication, fee preview, expired admin renewal, staff separation, Premium route restriction, mobile and desktop.",
  );
} catch (e) {
  console.error(
    page?.url(),
    await page?.evaluate(() => document.body.innerText),
  );
  throw e;
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
}
