import { paidMembership } from "./membership-fixture.mjs";
let staffRole = "admin";
import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile, mkdir } from "node:fs/promises";
import { resolve, sep, extname } from "node:path";
import { pathToFileURL } from "node:url";
const { default: puppeteer } = await import(
  pathToFileURL(process.env.PUPPETEER_MODULE).href
);
const dist = resolve("frontend/dist"),
  clients = new Set(),
  calls = [];
const tenant = "10000000-0000-4000-8000-000000000001",
  other = "10000000-0000-4000-8000-000000000002";
let inbox = [],
  campaigns = [],
  staff = true,
  receiptStatus = "pending";
let settings = {
  timezone: "America/Argentina/Buenos_Aires",
  schedule_enabled: false,
  paused: false,
  pause_until: null,
  pending_minutes: 15,
  prep_min: 20,
  prep_max: 35,
  periods: [],
  receiving: true,
};
const order = (id) => ({
  id,
  version: 1,
  status: "pending",
  buyer_name: "Comprador del barrio",
  channel: "web",
  created_at: new Date().toISOString(),
  total_due: "100.00",
  total_paid: "0.00",
  items: [],
  currency: "ARS",
  tenant_id: tenant,
});
const server = createServer(async (req, res) => {
  if (new URL(req.url, 'http://localhost').pathname === '/api/membership') { res.writeHead(200, {'Content-Type':'application/json'});res.end(JSON.stringify(paidMembership));return; }
  const u = new URL(req.url, "http://localhost"),
    p = u.pathname;
  if (p.startsWith("/api/")) {
    let body = "";
    for await (const chunk of req) body += chunk;
    body = body ? JSON.parse(body) : null;
    calls.push({ p, body, headers: req.headers });
    const reply = (data, code = 200) => {
      res.writeHead(code, { "Content-Type": "application/json" });
      res.end(JSON.stringify(data));
    };
    if (p === "/api/auth/refresh")
      return staff
        ? reply({
            token: "staff-fixture",
            user: {
              id: "staff",
              role: staffRole,
              brand_id: "brand",
              tenant_id: staffRole === "cashier" ? tenant : undefined,
            },
          })
        : reply({}, 401);
    if (p === "/api/tenants")
      return reply([
        { id: tenant, subdomain: "Centro", active: true },
        { id: other, subdomain: "Otra sucursal", active: true },
      ]);
    if (p === "/api/orders/events") {
      assert.equal(req.headers.authorization, "Bearer staff-fixture");
      assert.ok([tenant, other].includes(req.headers["x-tenant-id"]));
      assert.equal(u.search, "");
      res.writeHead(200, {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-store",
      });
      res.write("event: ready\ndata: {}\n\n");
      clients.add(res);
      req.on("close", () => clients.delete(res));
      return;
    }
    if (p === "/api/orders/inbox")
      return reply({ items: inbox, pending_count: inbox.length });
    if (p === "/api/orders") return reply({ items: inbox });
    if (p.match(/^\/api\/orders\/[^/]+\/payments$/))
      return reply({ items: [] });
    if (p.match(/^\/api\/orders\/[^/]+$/))
      return reply(order(p.split("/").at(-1)));
    if (
      p === "/api/pos/catalog" ||
      p === "/api/crm/customers" ||
      p === "/api/notifications"
    )
      return reply({ items: [] });
    if (p === "/api/ordering/settings") {
      if (body) settings = { ...settings, ...body };
      return reply(settings);
    }
    if (p === "/api/marketing/campaigns") {
      if (body) {
        const c = {
          id: "campaign-1",
          ...body,
          state: "draft",
          recipients: 0,
          delivered: 0,
          created_at: new Date().toISOString(),
        };
        campaigns.push(c);
        return reply(c, 201);
      }
      return reply({ items: campaigns, mail_configured: true });
    }
    if (p === "/api/marketing/campaigns/campaign-1")
      return reply({
        ...campaigns[0],
        eligible_count: 2,
        mail_configured: true,
        pending: 0,
        failed: 0,
      });
    if (p === "/api/marketing/campaigns/campaign-1/send") {
      campaigns[0].state = "queued";
      campaigns[0].recipients = 2;
      return reply(campaigns[0]);
    }
    if (p === "/api/public/newsletter/unsubscribe") {
      assert.equal(body.token, "u".repeat(43));
      return reply({ unsubscribed: true });
    }
    if (p === "/api/shopper/me") return reply({}, 401);
    if (p.endsWith("/buyer-session"))
      return reply({ creation_key: "key", payment_key: "payment" });
    if (
      p === "/api/public/buyer-order" ||
      p === "/api/public/buyer-order/cancel"
    ) {
      if (p.endsWith("/cancel")) {
        assert.equal(req.headers["x-buyer-tenant"], tenant);
        receiptStatus = "canceled";
      }
      return reply({
        ...order("order-buyer"),
        status: receiptStatus,
        business: "Burger del barrio",
        location: "Centro",
        fulfillment: "pickup",
        subtotal: "100.00",
        discount: "0.00",
        shipping: "0.00",
        payment_method: "at_local",
        online_available: false,
        pending_deadline: new Date(Date.now() + 900000).toISOString(),
        prep_min: 20,
        prep_max: 35,
        ended_reason:
          receiptStatus === "canceled"
            ? "Cancelado por el comprador antes de la confirmación"
            : "",
      });
    }
    if (p === "/api/public/buyer-order/events") {
      res.writeHead(503);
      res.end();
      return;
    }
    if (p.startsWith("/api/public/menu/"))
      return reply({
        location: {
          id: tenant,
          name: "Centro",
          brand_name: "Burger del barrio",
        },
        items: [],
        total: 0,
        categories: [],
        ordering_enabled: true,
        ordering_ready: true,
        extras: [],
        shipping_methods: [],
        payment_options: { at_local: true },
        currency: "ARS",
      });
    return reply({});
  }
  let file = resolve(dist, "." + p);
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
await mkdir(".cache/storefront-browser", { recursive: true });
const event = () => {
  for (const c of clients) c.write("event: changed\ndata: {}\n\n");
};
async function button(page, text) {
  await page.waitForFunction(
    (t) =>
      [...document.querySelectorAll("button")].some(
        (b) =>
          b.textContent.includes(t) &&
          !b.matches(":disabled") &&
          !b.closest("[inert]"),
      ),
    {},
    text,
  );
  await page.evaluate(
    (t) =>
      [...document.querySelectorAll("button")]
        .find(
          (b) =>
            b.textContent.includes(t) &&
            !b.matches(":disabled") &&
            !b.closest("[inert]"),
        )
        .click(),
    text,
  );
}
async function text(page, value) {
  await page.waitForFunction(
    (t) => document.body.textContent.includes(t),
    {},
    value,
  );
}
let checks = 0;
try {
  for (const width of [1440, 390])
    for (const theme of ["light", "dark"]) {
      staff = true;
      settings = {
        timezone: "America/Argentina/Buenos_Aires",
        schedule_enabled: false,
        paused: false,
        pause_until: null,
        pending_minutes: 15,
        prep_min: 20,
        prep_max: 35,
        periods: [],
        receiving: true,
      };
      inbox = [];
      campaigns = [];
      const sentBefore = calls.filter(
        (c) => c.p.endsWith("/send") && c.body,
      ).length;
      const page = await browser.newPage(),
        errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      page.on("dialog", (d) => d.accept());
      await page.setViewport({ width, height: 950 });
      await page.evaluateOnNewDocument(
        (t) => localStorage.setItem("legui-theme", t),
        theme,
      );
      await page.evaluateOnNewDocument(() => {
        window.chimes = 0;
        window.AudioContext = class {
          state = "suspended";
          currentTime = 0;
          destination = {};
          async resume() {
            this.state = "running";
          }
          createOscillator() {
            return {
              frequency: { value: 0 },
              connect() {},
              disconnect() {},
              start() {
                window.chimes++;
              },
              stop() {},
            };
          }
          createGain() {
            return {
              gain: {
                setValueAtTime() {},
                linearRampToValueAtTime() {},
                exponentialRampToValueAtTime() {},
              },
              connect() {},
              disconnect() {},
            };
          }
        };
      });

      staffRole = "cashier";
      await page.goto(origin + "/orders", { waitUntil: "domcontentloaded" });
      await text(page, "Pedidos en vivo");
      assert.equal(await page.evaluate(() => window.chimes), 0);
      checks++;
      await button(page, "Activar sonido");
      await page.waitForFunction(() =>
        document.querySelector("[aria-pressed=true]"),
      );
      inbox = [order("new-order-1")];
      event();
      await text(page, "Comprador del barrio");
      await page.waitForFunction(() => window.chimes === 2);
      checks++;
      event();
      await new Promise((r) => setTimeout(r, 200));
      assert.equal(await page.evaluate(() => window.chimes), 2);
      checks++;
      await button(page, "Revisar y confirmar");
      await text(page, "Motivo para rechazar o cancelar");
      await page.evaluate(() => {
        const e = [...document.querySelectorAll("label")].find((e) =>
          e.textContent.includes("Motivo para rechazar"),
        );
        const i = e.querySelector("input");
        i.value = "Sin disponibilidad";
        i.dispatchEvent(new Event("input", { bubbles: true }));
      });
      event();
      await new Promise((r) => setTimeout(r, 200));
      assert.equal(
        await page.$eval('input[maxlength="500"]', (e) => e.value),
        "Sin disponibilidad",
      );
      checks++;
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth + 1,
        ),
        false,
      );
      checks++;
      await page.evaluate(() => document.fonts.ready);
      // Poll active animations: a CSS transition can be replaced while a saved
      // finished promise stays pending in Chromium. Keep this wait bounded.
      await page.waitForFunction(() =>
        document
          .getAnimations()
          .every(
            (a) =>
              a.effect?.getComputedTiming().iterations === Infinity ||
              ["finished", "idle"].includes(a.playState),
          ),
      );
      await page.screenshot({
        path: `.cache/storefront-browser/inbox-${width}-${theme}.png`,
        fullPage: true,
      });
      staffRole = "admin";
      await page.goto(origin + "/store-operations", {
        waitUntil: "domcontentloaded",
      });
      await text(page, "Guardar atención del local");
      await button(page, "Agregar horario");
      await page.evaluate(() => {
        const i = document.querySelector('input[type="checkbox"]');
        i.checked = true;
        i.dispatchEvent(new Event("change", { bubbles: true }));
      });
      await button(page, "Guardar atención del local");
      await text(page, "Horarios y recepción de pedidos actualizados");
      const save = calls
        .filter((c) => c.p === "/api/ordering/settings" && c.body)
        .at(-1);
      assert.equal(save.body.paused, true);
      assert.ok(save.headers["idempotency-key"]);
      assert.equal(save.headers["x-tenant-id"], tenant);
      checks += 3;
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth + 1,
        ),
        false,
      );
      checks++;
      await page.evaluate(() => document.fonts.ready);
      // Poll active animations: a CSS transition can be replaced while a saved
      // finished promise stays pending in Chromium. Keep this wait bounded.
      await page.waitForFunction(() =>
        document
          .getAnimations()
          .every(
            (a) =>
              a.effect?.getComputedTiming().iterations === Infinity ||
              ["finished", "idle"].includes(a.playState),
          ),
      );
      await page.screenshot({
        path: `.cache/storefront-browser/hours-${width}-${theme}.png`,
        fullPage: true,
      });
      await page.goto(origin + "/campaigns", { waitUntil: "domcontentloaded" });
      await text(page, "Guardar borrador");
      await page.evaluate(() => {
        const i = document.querySelector('input[maxlength="120"]');
        i.value = "Nueva burger del finde";
        i.dispatchEvent(new Event("input", { bubbles: true }));
        const t = document.querySelector("textarea");
        t.value = "Este finde estrenamos una burger. Te esperamos.";
        t.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await button(page, "Guardar borrador");
      await text(page, "Nueva burger del finde");
      assert.equal(
        calls.filter((c) => c.p.endsWith("/send") && c.body).length,
        sentBefore,
      );
      checks++;
      await button(page, "Revisar");
      await text(page, "2 suscriptores elegibles");
      await page.evaluate(() => document.fonts.ready);
      // Poll active animations: a CSS transition can be replaced while a saved
      // finished promise stays pending in Chromium. Keep this wait bounded.
      await page.waitForFunction(() =>
        document
          .getAnimations()
          .every(
            (a) =>
              a.effect?.getComputedTiming().iterations === Infinity ||
              ["finished", "idle"].includes(a.playState),
          ),
      );
      await page.screenshot({
        path: `.cache/storefront-browser/campaign-${width}-${theme}.png`,
        fullPage: true,
      });
      await button(page, "Enviar campaña");
      await page.waitForSelector('[role="dialog"]');
      // Confirmation is the application's shared dialog.
      const dialog = await page.$('[role="dialog"]');
      if (dialog)
        await page.evaluate(() => {
          const d = document.querySelector('[role="dialog"]');
          [...d.querySelectorAll("button")]
            .find((b) => b.textContent.includes("Confirmar"))
            ?.click();
        });
      await text(page, "Campaña en cola");
      assert.equal(campaigns[0].state, "queued");
      checks++;
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth + 1,
        ),
        false,
      );
      checks++;
      staff = false;
      receiptStatus = "pending";
      await page.goto(origin + "/catalogo/" + tenant, {
        waitUntil: "domcontentloaded",
      });
      await text(page, "Cancelar pedido");
      await text(page, "Preparación estimada:");
      await button(page, "Cancelar pedido");
      await page.waitForSelector('[role="dialog"]');
      await page.evaluate(() => {
        const d = document.querySelector('[role="dialog"]');
        [...d.querySelectorAll("button")]
          .find((b) => b.textContent.includes("Confirmar"))
          .click();
      });
      await text(page, "Cancelado por el comprador");
      assert.equal(receiptStatus, "canceled");
      checks++;
      await page.goto(origin + "/baja#token=" + "u".repeat(43), {
        waitUntil: "domcontentloaded",
      });
      await text(page, "Confirmar baja");
      assert.equal(new URL(page.url()).hash, "");
      checks++;
      const before = calls.filter((c) => c.p.endsWith("/unsubscribe")).length;
      await button(page, "Confirmar baja");
      await text(page, "Suscripción cancelada");
      assert.equal(
        calls.filter((c) => c.p.endsWith("/unsubscribe")).length,
        before + 1,
      );
      checks++;
      assert.deepEqual(errors, []);
      checks++;
      await page.close();
    }
  console.log(
    `Storefront browser: ${checks} checks passed at desktop/mobile in light/dark: staff SSE/sound/no form reset, hours, reviewed campaigns, buyer cancellation and unsubscribe.`,
  );
} finally {
  for (const c of clients) c.end();
  await browser.close();
  server.closeAllConnections();
  await new Promise((r) => server.close(r));
}
