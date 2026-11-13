import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile, mkdir } from "node:fs/promises";
import { resolve, sep, extname } from "node:path";
import { pathToFileURL } from "node:url";
const { default: puppeteer } = await import(
  pathToFileURL(process.env.PUPPETEER_MODULE).href
);
const dist = resolve("frontend/dist"),
  calls = [];
const tenant = "10000000-0000-4000-8000-000000000001",
  product = "20000000-0000-4000-8000-000000000001",
  drink = "30000000-0000-4000-8000-000000000001",
  cheddar = "30000000-0000-4000-8000-000000000002";
let staffRole = "admin";
let staff = true,
  queue = 0,
  extras = [],
  config = {},
  settings = {},
  lastOrder;
const groups = () =>
  config.use_product_options
    ? config.groups.map((g, i) => ({
        ...g,
        id: "group-" + i,
        options: g.options.map((o) => {
          const e = extras.find((x) => x.ID === o.extra_id);
          return {
            id: e.ID,
            name: e.Name,
            price: String(o.price_override ?? e.CurrentPrice),
            available: e.IsActive,
          };
        }),
      }))
    : [];
const food = () => ({
  id: product,
  name: "Combo de la casa",
  description: "Burger con bebida a elección",
  price: "100.00",
  available: true,
  options_configured: config.use_product_options,
  option_groups: groups(),
  extras: extras.map((e) => ({
    id: e.ID,
    name: e.Name,
    price: String(e.CurrentPrice),
    available: true,
  })),
  category: "Combos",
});
const full = () =>
  settings.max_active_orders > 0 && queue >= settings.max_active_orders;
const delay = () =>
  Math.min(
    settings.max_queue_minutes || 0,
    queue * (settings.queue_step_minutes || 0),
  );
const load = () => ({
  active_count: queue,
  max_active_orders: settings.max_active_orders || 0,
  full: full(),
  prep_min: 20 + delay(),
  prep_max: 35 + delay(),
});
const server = createServer(async (req, res) => {
  const u = new URL(req.url, "http://localhost"),
    p = u.pathname;
  if (p.startsWith("/api/")) {
    let raw = "";
    for await (const part of req) raw += part;
    const body = raw ? JSON.parse(raw) : null;
    calls.push({ p, method: req.method, body, headers: req.headers });
    const reply = (data, code = 200) => {
      res.writeHead(code, { "Content-Type": "application/json" });
      res.end(JSON.stringify(data));
    };
    if (p === "/api/auth/refresh")
      return staff
        ? reply({
            token: "staff-options",
            user: {
              id: "staff",
              role: staffRole,
              brand_id: "brand",
              tenant_id: staffRole === "cashier" ? tenant : undefined,
            },
          })
        : reply({}, 401);
    if (p === "/api/tenants")
      return reply([{ id: tenant, subdomain: "Centro", active: true }]);
    if (p === "/api/notifications" || p === "/api/crm/customers")
      return reply({ items: [] });
    if (p === "/api/menu/availability")
      return reply({
        items: [
          {
            id: product,
            name: "Combo de la casa",
            enabled: true,
            in_stock: true,
            available: true,
          },
        ],
      });
    if (p === "/api/extras") {
      if (body) {
        const e = {
          ID: "new-extra",
          Name: body.name,
          CurrentPrice: body.current_price,
          IsActive: true,
        };
        extras.push(e);
        return reply(e, 201);
      }
      return reply(extras);
    }
    if (p.endsWith("/options")) {
      if (body) config = body;
      return reply({
        product_id: product,
        name: "Combo de la casa",
        ...config,
      });
    }
    if (p === "/api/ordering/settings") {
      if (body) settings = { ...settings, ...body };
      return reply({
        ...settings,
        active_count: queue,
        capacity_full: full(),
        receiving: !full(),
        estimated_prep_min: 20 + delay(),
        estimated_prep_max: 35 + delay(),
      });
    }
    if (p === "/api/shopper/me") return reply({}, 401);
    if (p.endsWith("/buyer-session"))
      return reply({ creation_key: "fresh-key", payment_key: "payment-key" });
    if (p === "/api/public/buyer-order") return reply({}, 404);
    if (p.endsWith("/quote")) {
      if (full())
        return reply(
          {
            message:
              "La cocina está al máximo de su capacidad. Esperá unos minutos",
          },
          409,
        );
      let total = 0;
      for (const line of body.items) {
        let unit = 100;
        for (const x of line.extras) {
          const opt = groups()
            .flatMap((g) => g.options)
            .find((o) => o.id === x.extra_id);
          unit += Number(opt.price) * x.quantity;
        }
        total += unit * line.quantity;
      }
      return reply({
        items: body.items,
        total: total.toFixed(2),
        subtotal: total.toFixed(2),
        shipping: "0.00",
        discount: "0.00",
        currency: "ARS",
        digest: "current-digest",
      });
    }
    if (p.startsWith("/api/public/menu/"))
      return reply({
        location: {
          id: tenant,
          name: "Centro",
          brand_name: "Burger del barrio",
        },
        currency: "ARS",
        ordering_enabled: true,
        ordering_ready: !full(),
        capacity_full: full(),
        prep_min: 20 + delay(),
        prep_max: 35 + delay(),
        items: [food()],
        total: 1,
        categories: ["Combos"],
        extras: food().extras,
        shipping_methods: [],
        payment_options: { at_local: true },
      });
    if (p === "/api/pos/catalog") return reply({ items: [food()] });
    if (p === "/api/orders/inbox")
      return reply({ items: [], pending_count: queue });
    if (p === "/api/orders") {
      if (body) {
        lastOrder = body;
        return reply({ id: "counter-order" }, 201);
      }
      return reply({ items: [] });
    }
    if (p === "/api/orders/events" || p.endsWith("/events"))
      return reply({}, 503);
    if (p === "/api/kitchen/load") return reply(load());
    if (p === "/api/kitchen")
      return reply({
        items: [
          {
            id: "kitchen-order",
            version: 1,
            status: "accepted",
            created_at: new Date().toISOString(),
            items: [
              {
                product_name: "Combo de la casa",
                quantity: 1,
                extras: [{ name: "Coca", quantity: 1, group_name: "Bebida" }],
              },
            ],
          },
        ],
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
await mkdir(".cache/options-browser", { recursive: true });
let checks = 0;
async function readyButton(page, t) {
  await page.waitForFunction(
    (t) =>
      [...document.querySelectorAll("button")].some(
        (b) =>
          b.textContent.includes(t) &&
          !b.matches(":disabled") &&
          !b.closest("[inert]"),
      ),
    {},
    t,
  );
}
async function button(page, t) {
  await page.waitForFunction(
    (t) =>
      [...document.querySelectorAll("button")].some(
        (b) =>
          b.textContent.includes(t) &&
          !b.matches(":disabled") &&
          !b.closest("[inert]"),
      ),
    {},
    t,
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
    t,
  );
}
async function text(page, t) {
  await page
    .waitForFunction((t) => document.body.textContent.includes(t), {}, t)
    .catch(async (e) => {
      console.error(
        "Expected text unavailable:",
        t,
        await page.evaluate(() => document.body.innerText.slice(-4500)),
      );
      await page.screenshot({
        path: ".cache/options-browser/failure.png",
        fullPage: true,
      });
      throw e;
    });
}
async function fillLabel(page, label, value) {
  await page.evaluate(
    (label, value) => {
      const l = [...document.querySelectorAll("label")].find((e) =>
        e.textContent.includes(label),
      );
      const el = l.querySelector("input");
      el.value = value;
      el.dispatchEvent(new Event("input", { bubbles: true }));
    },
    label,
    value,
  );
}
async function settle(page) {
  await page.evaluate(async () => {
    await document.fonts.ready;
    await Promise.allSettled(
      document
        .getAnimations()
        .filter((a) => a.effect?.getTiming().iterations !== Infinity)
        .map((a) => a.finished),
    );
  });
}
try {
  for (const width of [1440, 390])
    for (const theme of ["light", "dark"]) {
      staffRole = "admin";
      staff = true;
      queue = 0;
      lastOrder = null;
      extras = [
        { ID: drink, Name: "Coca", CurrentPrice: 25, IsActive: true },
        { ID: cheddar, Name: "Cheddar", CurrentPrice: 5, IsActive: true },
      ];
      config = { use_product_options: false, groups: [] };
      settings = {
        timezone: "America/Argentina/Buenos_Aires",
        schedule_enabled: false,
        paused: false,
        pause_minutes: 0,
        pending_minutes: 15,
        prep_min: 20,
        prep_max: 35,
        periods: [],
        max_active_orders: 0,
        queue_step_minutes: 0,
        max_queue_minutes: 60,
      };
      const page = await browser.newPage(),
        errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      page.on("dialog", (dialog) => dialog.accept());
      await page.setViewport({ width, height: width > 800 ? 768 : 900 });
      await page.evaluateOnNewDocument(
        (t) => localStorage.setItem("legui-theme", t),
        theme,
      );
      await page.goto(origin + "/product-options", {
        waitUntil: "domcontentloaded",
      });
      await text(page, "Combo de la casa");
      await page.evaluate((v) => {
        const s = [...document.querySelectorAll("select")].find((s) =>
          [...s.options].some((o) => o.value === v),
        );
        s.value = v;
        s.dispatchEvent(new Event("change", { bubbles: true }));
      }, product);
      await readyButton(page, "Guardar opciones del producto");
      await page.evaluate(() => {
        const c = document.querySelector(
          '.options-layout > div input[type="checkbox"]',
        );
        c.checked = true;
        c.dispatchEvent(new Event("change", { bubbles: true }));
      });
      await button(page, "Agregar grupo");
      await page.type('input[placeholder="Bebida"]', "Bebida");
      await fillLabel(page, "Mínimo de elecciones", "1");
      await button(page, "Agregar opción");
      await page.evaluate((v) => {
        const s = document.querySelector(".option-editor select");
        s.value = v;
        s.dispatchEvent(new Event("change", { bubbles: true }));
      }, drink);
      await page.type('input[placeholder="Usar precio del extra"]', "0");
      await button(page, "Guardar opciones del producto");
      await text(page, "Opciones del producto guardadas");
      await page.screenshot({
        path: `.cache/options-browser/manager-${width}-${theme}.png`,
        fullPage: true,
      });
      if (width > 800) {
        assert.ok(
          await page.$eval(
            ".options-layout > div > form",
            (e) => e.getBoundingClientRect().top < 280,
          ),
          "current product editor is visible beside the selector",
        );
        checks++;
      }
      assert.equal(config.groups[0].min, 1);
      assert.equal(config.groups[0].options[0].extra_id, drink);
      const saved = calls
        .filter((c) => c.p.endsWith("/options") && c.method === "PUT")
        .at(-1);
      assert.equal(saved.headers["x-tenant-id"], tenant);
      assert.ok(saved.headers["idempotency-key"]);
      checks += 4;
      // Extend the existing extra-backed model with an optional quantity group.
      config.groups.push({
        name: "Adicionales",
        min: 0,
        max: 2,
        options: [{ extra_id: cheddar, price_override: "10" }],
      });
      await page.goto(origin + "/store-operations", {
        waitUntil: "domcontentloaded",
      });
      await readyButton(page, "Guardar atención del local");
      await fillLabel(page, "Pedidos simultáneos admitidos", "2");
      await fillLabel(page, "Minutos extra por pedido en cola", "3");
      await fillLabel(page, "Tope de demora adicional", "6");
      await button(page, "Guardar atención del local");
      await text(page, "Horarios y recepción de pedidos actualizados");
      await page.screenshot({
        path: `.cache/options-browser/capacity-${width}-${theme}.png`,
        fullPage: true,
      });
      if (width > 800) {
        assert.ok(
          await page.$eval(
            ".ordering-sections",
            (e) => e.getBoundingClientRect().top < 360,
          ),
          "reception, capacity and hours appear together",
        );
        checks++;
      }
      assert.equal(settings.max_active_orders, 2);
      assert.equal(settings.queue_step_minutes, 3);
      checks += 2;
      staff = false;
      await page.goto(origin + "/catalogo/" + tenant, {
        waitUntil: "domcontentloaded",
      });
      await button(page, "Elegir producto");
      await page.waitForSelector('.product-option-choice input[type="radio"]');
      assert.equal(
        await page.$eval(".buyer-dialog button.btn-primary", (e) => e.disabled),
        true,
        "required choice prevents add",
      );
      checks++;
      await page.click(`input[type="radio"][value="${drink}"]`);
      await page.click(`input[type="checkbox"][value="${cheddar}"]`);
      await page.$eval(".product-option-quantity input", (e) => {
        e.value = "2";
        e.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await page.$eval('input[name="product-quantity"]', (e) => {
        e.value = "2";
        e.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await settle(page);
      await page.screenshot({
        path: `.cache/options-browser/choices-${width}-${theme}.png`,
        fullPage: true,
      });
      await button(page, "Sumar al pedido");
      await button(page, "Mi pedido");
      await text(page, "240");
      checks++;
      await button(page, "Cambiar opciones");
      await page.waitForFunction(
        () =>
          !document.querySelector(".buyer-sheet") &&
          !!document.querySelector('input[name="product-quantity"]'),
      );
      await settle(page);
      await page.click(`input[type="checkbox"][value="${cheddar}"]`);
      await button(page, "Guardar cambios");
      await text(page, "200");
      await button(page, "Revisar pedido");
      await text(page, "Total confirmado");
      const quoted = calls.filter((c) => c.p.endsWith("/quote")).at(-1).body;
      assert.equal(quoted.items[0].extras.length, 1);
      assert.equal(quoted.items[0].extras[0].extra_id, drink);
      assert.equal(quoted.items[0].quantity, 2);
      assert.ok(!("price" in quoted.items[0]));
      checks += 4;
      // A cart edit must fetch current product rules and remove choices no longer offered.
      await button(page, "Cambiar opciones");
      await page.waitForFunction(
        () =>
          !document.querySelector(".buyer-sheet") &&
          !!document.querySelector('input[name="product-quantity"]'),
      );
      await settle(page);
      await page.click(`input[type="checkbox"][value="${cheddar}"]`);
      await button(page, "Guardar cambios");
      await text(page, "220");
      config.groups = config.groups.filter((g) => g.name !== "Adicionales");
      await button(page, "Cambiar opciones");
      await page.waitForFunction(
        () =>
          !document.querySelector(".buyer-sheet") &&
          !!document.querySelector('input[name="product-quantity"]'),
      );
      await settle(page);
      await text(page, "Algunas opciones cambiaron");
      assert.equal(await page.$(`input[value="${cheddar}"]`), null);
      checks++;
      await button(page, "Guardar cambios");
      await text(page, "200");
      checks++;
      queue = 2;
      await page.goto(origin + "/catalogo/" + tenant, {
        waitUntil: "domcontentloaded",
      });
      await text(page, "La cocina está al máximo");
      assert.equal(await page.$(".menu-add"), null);
      checks++;
      const status = await page.evaluate(
        async (tenant) =>
          (
            await fetch(`/api/public/menu/${tenant}/quote`, {
              method: "POST",
              headers: { "Content-Type": "application/json" },
              body: JSON.stringify({ items: [] }),
            })
          ).status,
        tenant,
      );
      assert.equal(status, 409);
      checks++;
      queue = 1;
      await button(page, "Volver a consultar");
      await button(page, "Elegir producto");
      await text(page, "Bebida");
      checks++;
      staff = true;
      staffRole = "cashier";
      await page.goto(origin + "/kitchen", { waitUntil: "domcontentloaded" });
      await text(page, "Carga de cocina");
      await text(page, "23–38");
      await text(page, "Bebida: Coca");
      checks += 2;
      queue = 2;
      await button(page, "Actualizar");
      await text(page, "Cocina al máximo");
      checks++;
      queue = 0;
      config.groups.push({
        name: "Adicionales",
        min: 0,
        max: 2,
        options: [{ extra_id: cheddar, price_override: "10" }],
      });
      await page.goto(origin + "/orders", { waitUntil: "domcontentloaded" });
      await text(page, "Nuevo pedido de mostrador");
      await page.waitForFunction(
        (v) =>
          [...document.querySelectorAll("select")].some((s) =>
            [...s.options].some((o) => o.value === v),
          ),
        {},
        product,
      );
      await page.evaluate((v) => {
        const s = [...document.querySelectorAll("select")].find((s) =>
          [...s.options].some((o) => o.value === v),
        );
        s.value = v;
        s.dispatchEvent(new Event("change", { bubbles: true }));
      }, product);
      await page.waitForSelector(".product-option-choice input[type=radio]");
      assert.equal(
        await page.evaluate(
          () =>
            [...document.querySelectorAll("button")].find((b) =>
              b.textContent.includes("Sumar al pedido"),
            ).disabled,
        ),
        true,
      );
      checks++;
      await page.click(`input[type=radio][value="${drink}"]`);
      await button(page, "Sumar al pedido");
      await page.click(`input[type=radio][value="${drink}"]`);
      await page.click(`input[type=checkbox][value="${cheddar}"]`);
      await button(page, "Sumar al pedido");
      await button(page, "Crear pedido");
      await text(page, "Pedido creado");
      assert.equal(lastOrder.items.length, 2);
      assert.equal(lastOrder.items[0].extras.length, 1);
      assert.equal(lastOrder.items[1].extras.length, 2);
      checks += 3;
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth + 1,
        ),
        false,
      );
      assert.deepEqual(errors, []);
      checks += 2;
      await page.close();
    }
  console.log(
    `Options/capacity browser: ${checks} checks passed on desktop/mobile light/dark: scoped extra-backed setup, required/included choices, quantities/cart edits, server quote, kitchen settings/full/reopening and counter variants.`,
  );
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
}
