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

await mkdir(".cache/shopper-browser", { recursive: true });
const localA = "10000000-0000-4000-8000-000000000001",
  localB = "10000000-0000-4000-8000-000000000002";
let checks = 0;
async function button(page, label) {
  await page.waitForFunction(
    (t) =>
      [...document.querySelectorAll("button")].some(
        (e) => e.textContent.includes(t) && !e.disabled,
      ),
    {},
    label,
  );
  await page.evaluate(
    (t) =>
      [...document.querySelectorAll("button")]
        .find((e) => e.textContent.includes(t) && !e.disabled)
        .click(),
    label,
  );
}
try {
  for (const width of [1440, 390])
    for (const theme of ["light", "dark"]) {
      const page = await browser.newPage();
      const errors = [],
        calls = [];
      let logged = false,
        staff = false,
        savedCoupon = null,
        savedPhone = "",
        savedAddress = "";
      page.on("pageerror", (e) => errors.push(e.message));
      await page.setViewport({ width, height: 950 });
      await page.evaluateOnNewDocument(
        (t) => localStorage.setItem("legui-theme", t),
        theme,
      );
      await page.setRequestInterception(true);
      const reply = (r, data, status = 200) =>
        r.respond({
          status,
          contentType: "application/json",
          body: JSON.stringify(data),
        });
      page.on("request", (r) => {
        const path = new URL(r.url()).pathname;
        if (!path.startsWith("/api/")) return r.continue();
        const body = r.postData() ? JSON.parse(r.postData()) : null;
        calls.push({ path, body, headers: r.headers() });
        const profile = {
          id: "shopper-a",
          name: "Sofía",
          email: "sofia@example.invalid",
          email_verified: false,
          phone: savedPhone,
          delivery_address: savedAddress,
        };
        if (path === "/api/auth/refresh")
          return reply(
            r,
            staff
              ? {
                  token: "fixture-staff",
                  user: {
                    id: "admin-a",
                    role: "admin",
                    brand_id: "brand-a",
                    email: "admin@example.invalid",
                  },
                }
              : {},
            staff ? 200 : 401,
          );
        if (path === "/api/tenants")
          return reply(r, [
            {
              id: localA,
              subdomain: "City Bell",
              active: true,
              brand: { id: "brand-a", name: "Burger de barrio" },
            },
          ]);
        if (path === "/api/marketing/customers")
          return reply(r, {
            items: [
              {
                id: "shopper-a",
                name: "Sofía",
                orders: 2,
                newsletter_confirmed: false,
                personalization: true,
                segment: "Frecuente",
                paid_orders: 3,
                spent: "19000.00",
                favorite_product: "Doble de la casa",
              },
            ],
          });
        if (path === "/api/marketing/coupons") {
          if (r.method() === "POST") {
            savedCoupon = {
              id: "created-coupon",
              ...body,
              active: true,
              uses: 0,
              recipient: "Sofía",
              expires_at: new Date(Date.now() + 86400000).toISOString(),
            };
            return reply(
              r,
              { id: savedCoupon.id, code: savedCoupon.code },
              201,
            );
          }
          return reply(r, { items: savedCoupon ? [savedCoupon] : [] });
        }
        if (path.startsWith("/api/marketing/coupons/")) {
          savedCoupon.active = body.active;
          return reply(r, { id: savedCoupon.id, active: savedCoupon.active });
        }
        if (path === "/api/shopper/favorites") return reply(r, { items: [] });
        if (path === "/api/shopper/profile") {
          savedPhone = body.phone;
          savedAddress = body.delivery_address;
          return reply(r, { ok: true });
        }
        if (path === "/api/public/restaurants")
          return reply(r, {
            items: [
              {
                id: localA,
                business: "Burger de barrio",
                location: "City Bell",
                address: "Calle 123",
              },
              {
                id: localB,
                business: "Otra hamburguesería",
                location: "Centro",
                address: "Calle 456",
              },
            ],
          });
        if (path === "/api/shopper/me")
          return reply(r, logged ? profile : {}, logged ? 200 : 401);
        if (path === "/api/shopper/register" || path === "/api/shopper/login") {
          logged = true;
          return reply(r, profile);
        }
        if (path === "/api/public/shopper/logout") {
          logged = false;
          return reply(r, { ok: true });
        }
        if (path === "/api/shopper/orders")
          return reply(r, {
            items: [
              {
                id: "order-a",
                tenant_id: localA,
                business: "Burger de barrio",
                location: "City Bell",
                status: "accepted",
                total_due: "9000.00",
                total_paid: "0.00",
                created_at: new Date().toISOString(),
              },
              {
                id: "order-b",
                tenant_id: localB,
                business: "Otra hamburguesería",
                location: "Centro",
                status: "delivered",
                total_due: "8000.00",
                total_paid: "8000.00",
                created_at: new Date().toISOString(),
              },
            ],
          });
        if (path === "/api/shopper/coupons")
          return reply(r, {
            items: [
              {
                id: "coupon-a",
                code: "VOLVE10",
                percent: 10,
                minimum: "5000.00",
                business: "Burger de barrio",
                personal: true,
                location_id: localA,
                expires_at: new Date(Date.now() + 86400000).toISOString(),
                used: false,
              },
            ],
          });
        if (path === "/api/shopper/preferences")
          return reply(
            r,
            r.method() === "GET"
              ? {
                  items: [
                    {
                      id: "brand-a",
                      name: "Burger de barrio",
                      newsletter: false,
                    },
                  ],
                }
              : { ok: true },
          );
        if (path === "/api/shopper/claim" || path === "/api/shopper/password")
          return reply(r, { ok: true });
        if (path === "/api/shopper/resend")
          return reply(
            r,
            { message: "El envío de correo todavía no está configurado" },
            409,
          );
        if (path === "/api/public/buyer-order")
          return reply(r, {
            id: "order-a",
            tenant_id: localA,
            business: "Burger de barrio",
            location: "City Bell",
            status: "accepted",
            currency: "ARS",
            total_due: "9000.00",
            total_paid: "0.00",
            items: [{ name: "Doble de la casa", quantity: 1, extras: [] }],
            online_available: false,
            payment_method: "at_local",
            fulfillment: "pickup",
            account_payment_key: "80000000-0000-4000-8000-000000000001",
          });
        return reply(r, {}, 404);
      });
      await page.goto(origin + "/mi-cuenta?local=" + localA, {
        waitUntil: "networkidle0",
      });
      await page.waitForSelector(".shopper-auth form");
      await button(page, "Primera vez");
      await page.type("input[autocomplete=name]", "Sofía");
      await page.type("input[type=email]", "sofia@example.invalid");
      await page.type("input[type=password]", "Browser-Fixture-123!");
      await button(page, "Crear cuenta");
      await page.waitForSelector(".shopper-order");
      // An incomplete recovery link must retain the signed-in workspace.
      await page.goto(origin + "/mi-cuenta?accion=verify", {
        waitUntil: "networkidle0",
      });
      await page.waitForSelector(".shopper-order");
      await page.waitForFunction(() =>
        document
          .querySelector('[role="alert"]')
          ?.textContent.includes("El enlace está incompleto"),
      );
      assert.equal(await page.$(".shopper-auth"), null);
      checks++;
      await page.goto(origin + "/mi-cuenta?local=" + localA, {
        waitUntil: "networkidle0",
      });
      await page.waitForSelector(".shopper-order");
      assert.equal((await page.$$(".shopper-order")).length, 2);
      checks++;
      assert.ok(
        (await page.$eval(".shopper-content", (e) => e.textContent)).includes(
          "Otra hamburguesería",
        ),
      );
      checks++;
      assert.equal(
        await page.$eval(".shopper-sidebar nav > :first-child", (el) =>
          el.textContent.trim(),
        ),
        "Explorar restaurantes →",
      );
      checks++;
      await page.click(".shopper-start");
      await page.waitForSelector(".restaurant-grid article");
      await page.waitForFunction(() =>
        document.body.textContent.includes("Estás comprando como"),
      );
      checks++;
      await page.type(".directory-search input", "City Bell");
      await page.waitForFunction(
        () =>
          document.querySelectorAll(".restaurant-grid article").length === 1,
      );
      checks++;
      assert.ok(
        (await page.$eval(".restaurant-grid a", (el) => el.href)).includes(
          localA,
        ),
      );
      checks++;
      await page.goBack({ waitUntil: "domcontentloaded" });
      await page.waitForSelector(".shopper-order");
      await button(page, "Vincular el pedido");
      await page.waitForFunction(() =>
        document.body.textContent.includes("Pedido vinculado"),
      );
      checks++;
      await button(page, "Ver y seguir pedido");
      await page.waitForSelector(".buyer-receipt-status");
      assert.ok(
        calls.some(
          (c) =>
            c.path === "/api/public/buyer-order" &&
            c.headers["x-shopper-order"] === "order-a",
        ),
      );
      checks++;
      await page.click(".buyer-sheet .modal-close");
      await page.waitForSelector(".buyer-overlay", { hidden: true });
      await button(page, "Ver y seguir pedido");
      await page.waitForSelector(".buyer-receipt-status");
      await page.click(".buyer-sheet .modal-close");
      await page.waitForSelector(".buyer-overlay", { hidden: true });
      checks++;
      await button(page, "Mis cupones");
      await page.waitForSelector(".shopper-coupon");
      assert.equal(
        await page.$eval(".shopper-coupon a", (e) =>
          new URL(e.href).searchParams.get("cupon"),
        ),
        "VOLVE10",
      );
      checks++;
      await button(page, "Novedades de marcas");
      await page.waitForSelector(".shopper-preference input");
      await page.waitForSelector(".shopper-preference input:not(:disabled)");
      await page.click(".shopper-preference input");
      await page.waitForFunction(() =>
        document.body.textContent.includes("Preferencias guardadas"),
      );
      assert.ok(
        calls.some(
          (c) =>
            c.path === "/api/shopper/preferences" &&
            c.body?.newsletter === true,
        ),
      );
      checks++;
      assert.ok(
        (await page.$eval(".shopper-content", (e) => e.textContent)).includes(
          "confirmar tu dirección",
        ),
      );
      checks++;
      await page.click("[data-personalization]");
      await page.waitForFunction(
        () =>
          document.querySelector("[data-personalization]").checked &&
          document.body.textContent.includes("Preferencias guardadas"),
      );
      assert.ok(
        calls.some(
          (c) =>
            c.path === "/api/shopper/preferences" &&
            c.body?.personalization === true,
        ),
      );
      checks++;
      await button(page, "Mi perfil");
      await page.type('input[autocomplete="tel"]', "1155554444");
      await page.type(
        'textarea[autocomplete="street-address"]',
        "Calle habitual 123",
      );
      await button(page, "Guardar datos");
      await page.waitForFunction(() =>
        document.body.textContent.includes("Datos guardados"),
      );
      assert.ok(
        calls.some(
          (c) =>
            c.path === "/api/shopper/profile" && c.body?.phone === "1155554444",
        ),
      );
      checks++;
      await button(page, "Enviar confirmación");
      await page.waitForFunction(() =>
        document.body.textContent.includes(
          "El envío de correo todavía no está configurado",
        ),
      );
      checks++;
      assert.ok(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      );
      checks++;
      await page.screenshot({
        path: `.cache/shopper-browser/${width}-${theme}.png`,
        fullPage: true,
      });
      await button(page, "Cerrar sesión");
      await page.waitForSelector(".shopper-auth");
      const stored = await page.evaluate(() =>
        JSON.stringify({ ...sessionStorage, ...localStorage }),
      );
      assert.ok(
        !stored.includes("sofia@example.invalid") &&
          !stored.includes("Browser-Fixture"),
      );
      checks++;
      staff = true;
      await page.goto(origin + "/marketing", { waitUntil: "networkidle0" });
      await page.waitForSelector('input[placeholder="VOLVE10"]');
      await page.type('input[placeholder="VOLVE10"]', "VOLVE10");
      await page.select("form select", "shopper-a");
      await page.click(".marketing-check input");
      await button(page, "Crear cupón");
      await page.waitForSelector(".marketing-coupon");
      const couponRequest = calls.find(
        (c) =>
          c.path === "/api/marketing/coupons" && c.body?.code === "VOLVE10",
      );
      assert.equal(couponRequest.body.shopper_id, "shopper-a");
      checks++;
      assert.equal(couponRequest.body.all_locations, true);
      checks++;
      assert.equal(couponRequest.headers["x-tenant-id"], localA);
      checks++;
      assert.ok(couponRequest.headers["idempotency-key"]);
      checks++;
      await button(page, "Pausar");
      await page.waitForFunction(() =>
        [...document.querySelectorAll(".marketing-coupon button")].some((b) =>
          b.textContent.includes("Activar"),
        ),
      );
      checks++;
      assert.deepEqual(errors, []);
      checks++;
      await page.close();
    }
  console.log(
    `Shopper account browser: ${checks} checks passed on desktop/mobile, light/dark.`,
  );
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
}
