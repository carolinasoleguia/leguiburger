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
  zoneId = "30000000-0000-4000-8000-000000000001",
  orderId = "40000000-0000-4000-8000-000000000001";
let activePage;
let staffRole = "admin";
let staff = true,
  zones = [],
  zoned = false,
  saved = [],
  paused = false,
  stock = true,
  failSave = true;
const food = () => ({
  id: product,
  name: "Doble de la casa",
  description: "Doble carne y cheddar",
  price: "120.00",
  available: !paused && stock,
  category: "Hamburguesas",
  image_url: "",
});
const profile = {
  id: "shopper-1",
  name: "Sofía",
  email: "sofia@example.invalid",
  email_verified: true,
  phone: "1155550123",
  delivery_address: "Calle de la casa 123",
};
const server = createServer(async (req, res) => {
  const u = new URL(req.url, "http://localhost"),
    p = u.pathname;
  if (p.startsWith("/api/")) {
    let raw = "";
    for await (const chunk of req) raw += chunk;
    const body = raw ? JSON.parse(raw) : null;
    calls.push({ p, body, headers: req.headers, method: req.method });
    const reply = (data, code = 200) => {
      res.writeHead(code, { "Content-Type": "application/json" });
      res.end(JSON.stringify(data));
    };
    if (p === "/api/auth/refresh")
      return staff
        ? reply({
            token: "staff-token",
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
    if (p === "/api/notifications") return reply({ items: [] });
    if (p === "/api/shopper/me") return reply(profile);
    if (p === "/api/shopper/orders")
      return reply({
        items: [
          {
            id: orderId,
            tenant_id: tenant,
            business: "Burger de barrio",
            location: "Centro",
            status: "delivered",
            total_due: "100.00",
            total_paid: "100.00",
            created_at: new Date().toISOString(),
          },
        ],
      });
    if (p === "/api/shopper/coupons" || p === "/api/shopper/preferences")
      return reply({ items: [] });
    if (p === "/api/shopper/favorites") return reply({ items: saved });
    if (p.startsWith("/api/shopper/favorites/")) {
      if (req.method === "PUT")
        saved = [
          {
            ...food(),
            product_id: product,
            tenant_id: tenant,
            business: "Burger de barrio",
            location: "Centro",
          },
        ];
      else saved = [];
      return reply({ ok: true });
    }
    if (p.endsWith("/repeat"))
      return reply({
        tenant_id: tenant,
        source_order_id: orderId,
        items: [
          {
            product_id: product,
            name: "Doble de la casa",
            price: "120.00",
            quantity: 1,
            note: "Sin cebolla",
            extras: [
              {
                id: "extra-cheddar",
                name: "Cheddar",
                price: "10.00",
                quantity: 2,
              },
            ],
          },
        ],
        changes: [
          "Doble de la casa: el precio actual es diferente al de tu compra anterior.",
          "Papas: no están disponibles y no se agregaron.",
        ],
        notice:
          "Revisá tu pedido con los precios actuales. Los cupones y el envío anteriores no se vuelven a aplicar.",
      });
    if (p === "/api/delivery/settings") {
      if (body) zoned = body.zones_enabled;
      return reply({ zones_enabled: zoned });
    }
    if (p === "/api/delivery/zones") {
      if (body) {
        if (failSave) {
          failSave = false;
          return reply({ message: "No se pudo guardar. Reintentá." }, 503);
        }
        zones = [{ id: zoneId, ...body }];
        return reply(zones[0], 201);
      }
      return reply({ items: zones });
    }
    if (p === "/api/menu/availability")
      return reply({
        items: [
          {
            id: product,
            name: "Doble de la casa",
            enabled: !paused,
            in_stock: stock,
            available: !paused && stock,
          },
        ],
      });
    if (p.endsWith("/availability") && body) {
      paused = !body.enabled;
      return reply({ id: product, enabled: body.enabled });
    }
    if (p.endsWith("/ticket")) {
      const base = {
        id: orderId,
        kind: u.searchParams.get("kind"),
        business: "Burger <script>alert(1)</script>",
        location: "Centro",
        status: "accepted",
        created_at: new Date().toISOString(),
        fulfillment: "delivery",
        items: [
          {
            name: "Doble de la casa",
            quantity: 1,
            note: "Sin cebolla",
            extras: [{ name: "Cheddar", quantity: 2 }],
          },
        ],
      };
      return reply(
        base.kind === "kitchen"
          ? base
          : {
              ...base,
              name: "Sofía",
              phone: "1155550123",
              address: "Calle de la casa 123",
              zone: "Centro",
              postal_code: "1896",
              currency: "ARS",
              total_due: "160.00",
              total_paid: "0.00",
              shipping: "20.00",
              discount: "0.00",
            },
      );
    }
    if (p.endsWith("/buyer-session"))
      return reply({ creation_key: "fresh-key", payment_key: "payment-key" });
    if (p === "/api/public/buyer-order") return reply({}, 404);
    if (p.endsWith("/orders") && req.method === "POST")
      return reply(
        {
          id: orderId,
          tenant_id: tenant,
          status: "pending",
          currency: "ARS",
          total_due: "148.00",
          total_paid: "0.00",
          fulfillment: body.fulfillment,
          items: [],
          created_at: new Date().toISOString(),
        },
        201,
      );
    if (p.endsWith("/quote")) {
      if (body.fulfillment === "delivery" && body.postal_code !== "1896")
        return reply(
          {
            message:
              "No tenemos cobertura para ese código postal en la zona elegida",
          },
          409,
        );
      return reply({
        items: body.items,
        total: body.coupon_code === "VOLVE10" ? "148.00" : "160.00",
        subtotal: "140.00",
        shipping: "20.00",
        shipping_name: "Centro",
        discount: body.coupon_code === "VOLVE10" ? "12.00" : "0.00",
        currency: "ARS",
        digest: "fresh-quote",
      });
    }
    if (p.startsWith("/api/public/menu/"))
      return reply({
        location: {
          id: tenant,
          name: "Centro",
          brand_name: "Burger de barrio",
        },
        items: [food()],
        total: 1,
        categories: ["Hamburguesas"],
        ordering_enabled: true,
        ordering_ready: true,
        delivery_zones_enabled: zoned,
        delivery_zones: zones.map((z) => ({ ...z, price: z.cost })),
        shipping_methods: [],
        extras: [
          {
            id: "extra-cheddar",
            name: "Cheddar",
            price: "10.00",
            available: true,
          },
        ],
        payment_options: { at_local: true },
        currency: "ARS",
      });
    if (p === "/api/orders/events" || p === "/api/public/buyer-order/events")
      return reply({}, 503);
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
await mkdir(".cache/commerce-browser", { recursive: true });
let checks = 0;
async function text(page, t) {
  await page.waitForFunction(
    (t) => document.body.textContent.includes(t),
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
      const creationsBefore = calls.filter(
        (c) => c.p.endsWith("/orders") && c.method === "POST",
      ).length;
      staffRole = "admin";
      staff = true;
      zones = [];
      zoned = false;
      saved = [];
      paused = false;
      stock = true;
      failSave = true;
      const page = (activePage = await browser.newPage()),
        errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.setViewport({ width, height: 950 });
      await page.evaluateOnNewDocument((t) => {
        localStorage.setItem("legui-theme", t);
        window.printCount = 0;
        window.print = () => window.printCount++;
      }, theme);
      await page.goto(origin + "/delivery-zones", {
        waitUntil: "domcontentloaded",
      });
      await button(page, "Guardar zona").catch(() => {});
      // Fill explicitly; submitting an empty native form must not call the API.
      assert.equal(
        calls.filter((c) => c.p === "/api/delivery/zones" && c.body).length % 2,
        0,
      );
      await page.evaluate(() => {
        const fill = (s, v) => {
          const e = document.querySelector(s);
          e.value = v;
          e.dispatchEvent(new Event("input", { bubbles: true }));
        };
        fill('input[placeholder="City Bell centro"]', "Centro");
        fill('input[placeholder="1896, 1897"]', "1896, 1897");
        const nums = document.querySelectorAll('input[inputmode="decimal"]');
        for (const [i, v] of ["20.00", "100.00"].entries()) {
          nums[i].value = v;
          nums[i].dispatchEvent(new Event("input", { bubbles: true }));
        }
      });
      await button(page, "Guardar zona");
      await text(page, "No se pudo guardar");
      assert.equal(
        await page.$eval(
          'input[placeholder="City Bell centro"]',
          (e) => e.value,
        ),
        "Centro",
      );
      checks++;
      await button(page, "Guardar zona");
      await text(page, "Zona de envío guardada");
      const attempts = calls
        .filter((c) => c.p === "/api/delivery/zones" && c.body)
        .slice(-2);
      assert.equal(
        attempts[0].headers["idempotency-key"],
        attempts[1].headers["idempotency-key"],
      );
      assert.equal(attempts[1].headers["x-tenant-id"], tenant);
      checks += 2;
      await button(page, "Habilitar cobertura");
      await text(page, "Cobertura por código postal activa");
      checks++;
      await settle(page);
      await page.screenshot({
        path: `.cache/commerce-browser/delivery-${width}-${theme}.png`,
        fullPage: true,
      });
      await page.goto(origin + "/menu-availability", {
        waitUntil: "domcontentloaded",
      });
      await text(page, "Doble de la casa");
      await button(page, "Pausar producto");
      await text(page, "Pausado por el local");
      assert.equal(paused, true);
      checks++;
      stock = false;
      await button(page, "Reactivar producto");
      await text(page, "Agotado según inventario");
      assert.equal(paused, false);
      checks++;
      stock = true;
      staff = false;
      await page.goto(origin + "/catalogo/" + tenant, {
        waitUntil: "domcontentloaded",
      });
      await button(page, "Guardar favorito");
      await text(page, "Quitar de favoritos");
      assert.equal(saved.length, 1);
      checks++;
      await page.goto(origin + "/mi-cuenta", { waitUntil: "domcontentloaded" });
      await text(page, "Mis pedidos");
      await button(page, "Mis favoritos");
      await text(page, "Doble de la casa");
      const favoriteLink = await page.$eval('a[href*="producto="]', (e) =>
        e.getAttribute("href"),
      );
      assert.ok(favoriteLink.includes(product));
      checks++;
      await page.goto(origin + favoriteLink, { waitUntil: "domcontentloaded" });
      await page.waitForSelector(
        '.buyer-dialog textarea[placeholder="Por ejemplo, sin cebolla"]',
      );
      checks++;
      await page.goto(origin + "/catalogo/" + tenant + "?repetir=" + orderId, {
        waitUntil: "domcontentloaded",
      });
      await text(page, "Volvé a disfrutarlo");
      await text(page, "precio actual es diferente");
      await text(page, "Papas: no están disponibles");
      await text(page, "Sin cebolla");
      checks += 2;
      assert.equal(
        await page.evaluate(() =>
          [...document.querySelectorAll(".buyer-total strong")].some((e) =>
            e.textContent.includes("140"),
          ),
        ),
        true,
        "repeat includes two extra units in estimate",
      );
      checks++;
      assert.equal(
        calls.filter((c) => c.p.endsWith("/orders") && c.method === "POST")
          .length,
        creationsBefore,
        "repeat must not create a purchase",
      );
      checks++;
      await page.evaluate(() => {
        const e = document.querySelector(
          'input[type="radio"][value="delivery"]',
        );
        e.click();
      });
      await page.waitForSelector('input[name="delivery-postal-code"]');
      assert.ok(
        (await page.$eval(
          'input[name="delivery-postal-code"]',
          (e) => e.getBoundingClientRect().width,
        )) >= 200,
        "postal field has usable touch width",
      );
      checks++;
      await page.type(".buyer-coupon input", "VOLVE10");
      await page.type('input[name="delivery-postal-code"]', "9999");
      await button(page, "Revisar pedido");
      await text(page, "No tenemos cobertura");
      assert.equal(
        await page.evaluate(() =>
          [...document.querySelectorAll("button")].some((b) =>
            b.textContent.includes("Confirmar pedido"),
          ),
        ),
        false,
      );
      checks++;
      await page.$eval('input[name="delivery-postal-code"]', (e) => {
        e.value = "1896";
        e.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await button(page, "Revisar pedido");
      await text(page, "Total confirmado");
      const quote = calls.filter((c) => c.p.endsWith("/quote")).at(-1).body;
      assert.equal(quote.delivery_zone_id, zoneId);
      assert.equal(quote.postal_code, "1896");
      assert.equal(quote.items[0].extras[0].quantity, 2);
      assert.ok(!("shipping_method_id" in quote));
      checks += 4;
      assert.equal(quote.coupon_code, "VOLVE10");
      assert.deepEqual(
        await page.evaluate(() => ({
          name: document.querySelector('input[name="buyer-name"]').value,
          email: document.querySelector('input[name="buyer-email"]').value,
          phone: document.querySelector('input[name="buyer-phone"]').value,
          address: document.querySelector('textarea[name="buyer-address"]')
            .value,
        })),
        {
          name: profile.name,
          email: profile.email,
          phone: profile.phone,
          address: profile.delivery_address,
        },
      );
      await page.click(".buyer-remember input");
      await settle(page);
      await page.screenshot({
        path: `.cache/commerce-browser/repeat-${width}-${theme}.png`,
        fullPage: true,
      });
      await button(page, "Confirmar pedido");
      await page.waitForSelector(".buyer-receipt-status");
      const created = calls
        .filter((c) => c.p.endsWith("/orders") && c.method === "POST")
        .at(-1).body;
      assert.equal(created.coupon_code, "VOLVE10");
      assert.equal(created.remember_contact, true);
      assert.equal(created.name, profile.name);
      assert.ok(
        !("shopper_id" in created),
        "account identity is resolved by the server cookie",
      );
      checks += 6;
      staff = true;
      staffRole = "cashier";
      for (const kind of ["kitchen", "receipt"]) {
        await page.goto(
          origin + `/orders/${orderId}/print?tenant=${tenant}&kind=${kind}`,
          { waitUntil: "domcontentloaded" },
        );
        await page.waitForSelector(".ticket-paper");
        await text(page, "Sin cebolla");
        assert.equal(
          await page.$(".ticket-paper script"),
          null,
          "ticket text is escaped",
        );
        assert.equal(await page.evaluate(() => window.printCount), 0);
        checks += 2;
        if (kind === "kitchen")
          assert.equal(
            await page.evaluate(() =>
              document
                .querySelector(".ticket-paper")
                .textContent.includes("Sofía"),
            ),
            false,
          );
        else await text(page, "Recibido:");
        await button(
          page,
          kind === "kitchen" ? "Imprimir comanda" : "Imprimir comprobante",
        );
        assert.equal(await page.evaluate(() => window.printCount), 1);
        checks++;
        await page.emulateMediaType("print");
        assert.equal(
          await page.$eval(
            ".ticket-controls",
            (e) => getComputedStyle(e).display,
          ),
          "none",
        );
        assert.ok(
          (await page.$eval(
            ".ticket-paper",
            (e) => e.getBoundingClientRect().width,
          )) <= 273,
        );
        checks += 2;
        await page.emulateMediaType("screen");
      }
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
    `Commerce browser: ${checks} checks passed on desktop/mobile in light/dark: scoped zone setup/retry, ingredient availability, favorites, repeat changes/current quote, coverage rejection and safe 80mm print projection.`,
  );
} catch (error) {
  if (activePage && !activePage.isClosed()) {
    console.error(
      activePage.url(),
      await activePage.$eval("body", (e) => e.innerText.slice(-3000)),
      calls.slice(-10),
    );
    await activePage.screenshot({
      path: ".cache/commerce-experience-failure.png",
      fullPage: true,
    });
  }
  throw error;
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
}
