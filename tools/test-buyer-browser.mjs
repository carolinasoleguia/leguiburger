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

await mkdir(".cache/buyer-browser", { recursive: true });
const tenant = "10000000-0000-4000-8000-000000000001",
  otherTenant = "10000000-0000-4000-8000-000000000002";
const product = "20000000-0000-4000-8000-000000000001",
  extra = "30000000-0000-4000-8000-000000000001",
  shipping = "40000000-0000-4000-8000-000000000001";
let assertions = 0;
const check = (test, message) => {
  assert.ok(test, message);
  assertions++;
};
async function button(page, text) {
  await page.waitForFunction(
    (t) =>
      [...document.querySelectorAll("button")].some(
        (x) =>
          x.textContent.trim().includes(t) &&
          x.getClientRects().length &&
          !x.disabled,
      ),
    {},
    text,
  );
  await page.evaluate(
    (t) =>
      [...document.querySelectorAll("button")]
        .find(
          (x) =>
            x.textContent.trim().includes(t) &&
            x.getClientRects().length &&
            !x.disabled,
        )
        .click(),
    text,
  );
}
try {
  for (const width of [1440, 390])
    for (const theme of ["light", "dark"]) {
      const page = await browser.newPage();
      const errors = [],
        calls = [];
      let order = null,
        creates = 0,
        quotes = 0,
        blocked = true,
        humanVerified = false,
        stale = true,
        unknown = true,
        online = false,
        status = "pending",
        holdQuote = false,
        release;
      await page.setViewport({ width, height: 950 });
      await page.evaluateOnNewDocument(
        (t) => localStorage.setItem("legui-theme", t),
        theme,
      );
      await page.evaluateOnNewDocument(() => {
        window.turnstile = {
          render(target, options) {
            const button = document.createElement("button");
            button.type = "button";
            button.textContent = "Completar verificación de prueba";
            button.onclick = () => options.callback("fixture-human-proof");
            target.appendChild(button);
            return 1;
          },
          remove() {},
          reset() {},
        };
      });
      page.on("pageerror", (e) => errors.push(e.message));
      await page.setRequestInterception(true);
      const reply = (r, data, status = 200) =>
        r.respond({
          status,
          contentType: "application/json",
          body: JSON.stringify(data),
        });
      const receipt = () => ({ ...order, status, online_available: online });
      page.on("request", (r) => {
        const path = new URL(r.url()).pathname;
        if (!path.startsWith("/api/")) return r.continue();
        const body = r.postData() ? JSON.parse(r.postData()) : undefined;
        calls.push({ path, body, headers: r.headers(), method: r.method() });
        if (path === "/api/public/restaurants/leguiburger/centro")
          return reply(r, {
            tenant_id: tenant,
            path: "/leguiburger/centro/carta",
          });
        if (path === "/api/auth/refresh") return reply(r, {}, 401);
        if (path.endsWith("/human-check")) {
          if (r.method() === "GET")
            return reply(r, {
              configured: theme === "dark",
              site_key: "fixture-site",
              action: "checkout",
              cdata: "fixture-context",
            });
          assert.equal(body.token, "fixture-human-proof");
          humanVerified = true;
          return reply(r, { verified: true });
        }
        if (path.endsWith("/buyer-session"))
          return reply(r, {
            creation_key: "60000000-0000-4000-8000-000000000001",
            payment_key: "70000000-0000-4000-8000-000000000001",
          });
        if (path.endsWith("/quote")) {
          quotes++;
          const q = {
            items: [],
            subtotal: "220.50",
            shipping: "20.50",
            shipping_name: "Zona centro",
            total: "241.00",
            currency: "ARS",
            digest: "a".repeat(64),
          };
          if (holdQuote) {
            release = () => reply(r, q);
            return;
          }
          return reply(r, q);
        }
        if (path.endsWith("/orders")) {
          creates++;
          if (blocked) {
            blocked = false;
            return reply(
              r,
              {
                code: "CHECKOUT_RATE_LIMITED",
                message:
                  "Detectamos demasiados intentos de compra. Esperá unos minutos y volvé a intentar.",
              },
              429,
            );
          }
          if (!humanVerified)
            return reply(
              r,
              {
                code: "HUMAN_VERIFICATION_REQUIRED",
                message:
                  "Necesitamos una verificación adicional antes de confirmar tu compra.",
              },
              403,
            );
          if (stale) {
            stale = false;
            return reply(
              r,
              {
                message:
                  "La carta o el precio cambiaron. Revisá el resumen actualizado.",
              },
              409,
            );
          }
          order = {
            id: "50000000-0000-4000-8000-000000000001",
            tenant_id: tenant,
            status,
            currency: "ARS",
            total_due: "241.00",
            total_paid: "0.00",
            fulfillment: "delivery",
            payment_method: "at_local",
            business: "Burger de barrio",
            location: "Local Centro",
            shipping: "20.50",
            items: [
              {
                name: "Doble de la casa",
                quantity: 2,
                note: "Sin cebolla",
                extras: [{ name: "Cheddar", quantity: 1 }],
              },
            ],
            tracking_path: "/order#token=" + "b".repeat(43),
          };
          if (unknown) {
            unknown = false;
            return reply(r, { message: "Sin confirmación" }, 500);
          }
          return reply(r, receipt(), 201);
        }
        if (path === "/api/public/buyer-order")
          return order ? reply(r, receipt()) : reply(r, {}, 404);
        if (path.startsWith("/api/public/menu/"))
          return reply(r, {
            location: {
              id: tenant,
              name: "Local Centro",
              brand_name: "Burger de barrio",
              address: "Av. de la cocina 125",
              hours: "Martes a domingo, 19 a 23",
              phone: "11 5555 0123",
            },
            currency: "ARS",
            ordering_enabled: !path.includes(otherTenant),
            payment_options: { at_local: true, mercadopago: false },
            extras: [
              { id: extra, name: "Cheddar", price: "10.25", available: true },
            ],
            shipping_methods: [
              { id: shipping, name: "Zona centro", price: "20.50" },
            ],
            categories: ["Hamburguesas"],
            total: 2,
            items: [
              {
                id: product,
                name: "Doble de la casa",
                price: "100.00",
                description: "Doble carne, cheddar y salsa de la casa",
                available: true,
                category: "Hamburguesas",
                image_url: "",
              },
              {
                id: "unavailable",
                name: "Especial agotada",
                price: "150.00",
                available: false,
                category: "Hamburguesas",
                image_url: "",
              },
            ],
          });
        return reply(r, {}, 404);
      });
      await page.goto(
        origin +
          (theme === "dark"
            ? "/leguiburger/centro/carta"
            : "/catalogo/" + tenant),
        {
          waitUntil: "networkidle0",
        },
      );
      await page.waitForSelector(".menu-add:not(:disabled)");
      check(
        !calls.some((x) => x.path.startsWith("/api/auth/")),
        "public buying never requests an app session",
      );
      check(
        (await page.$$(".menu-add:disabled")).length === 1,
        "unavailable cannot be ordered",
      );
      await button(page, "Elegir producto");
      await page.waitForSelector(".buyer-extra input");
      await page.click(".buyer-extra input");
      await page.$eval(".buyer-dialog input[name=product-quantity]", (el) => {
        el.value = "2";
        el.dispatchEvent(new Event("input", { bubbles: true }));
      });
      await page.type(".buyer-dialog textarea", "Sin cebolla");
      await button(page, "Sumar al pedido");
      await button(page, "Mi pedido");
      check(
        (await page.$eval(".buyer-sheet", (el) => el.textContent)).includes(
          "2 × Doble de la casa",
        ),
        "cart quantity",
      );
      await page.click("input[value=delivery]");
      holdQuote = true;
      await button(page, "Revisar pedido");
      await page.waitForFunction(
        () =>
          document.querySelector(".buyer-sheet").getAttribute("aria-busy") ===
          "true",
      );
      check(
        await page.$eval(".buyer-sheet .btn-primary", (el) => el.disabled),
        "quote loading disables action",
      );
      holdQuote = false;
      await release();
      await page.waitForSelector("input[name=buyer-name]");
      await page.type("input[name=buyer-name]", "Cliente de prueba");
      await page.type("input[name=buyer-phone]", "11 5555 0123");
      await page.type("input[name=buyer-email]", "buyer@example.invalid");
      await page.type("textarea[name=buyer-address]", "Calle de prueba 123");
      check(
        await page.$$eval(
          ".buyer-sheet option",
          (els) => !els.some((x) => x.value === "mercadopago"),
        ),
        "unconfigured MP is hidden",
      );
      await button(page, "Confirmar pedido");
      await page.waitForFunction(() =>
        document
          .querySelector(".buyer-sheet .error-msg")
          ?.textContent.includes("demasiados intentos"),
      );
      check(
        (await page.$("input[name=buyer-name]")) !== null,
        "temporary block preserves checkout contact",
      );
      await button(page, "Reintentar la misma confirmación");
      await page.waitForSelector(".buyer-human-check");
      check(
        (await page.$eval('input[name="buyer-name"]', (e) => e.value)) ===
          "Cliente de prueba",
        "human check preserves contact and cart",
      );
      if (theme === "dark") {
        await button(page, "Completar verificación de prueba");
        await button(page, "Verificar y continuar");
      } else {
        await page.waitForFunction(() =>
          document
            .querySelector(".buyer-human-check")
            ?.textContent.includes("no está disponible"),
        );
        humanVerified = true; // Simulate expiry of risk window / an owner releasing a false positive.
        await button(page, "Volver a intentar la compra");
      }

      await page.waitForFunction(() =>
        document
          .querySelector(".buyer-sheet .error-msg")
          ?.textContent.includes("cambiaron"),
      );
      check(
        !(await page.$("input[name=buyer-name]")),
        "stale quote requires fresh review",
      );
      await button(page, "Revisar pedido");
      await page.waitForSelector("input[name=buyer-name]");
      check(
        (await page.$eval("input[name=buyer-name]", (el) => el.value)) ===
          "Cliente de prueba",
        "contact draft survives stale quote",
      );
      await button(page, "Confirmar pedido");
      await page.waitForFunction(() =>
        document
          .querySelector(".buyer-sheet .error-msg")
          ?.textContent.includes("Sin confirmación"),
      );
      await button(page, "Reintentar la misma confirmación");
      await page.waitForSelector(".buyer-receipt-status");
      const attempts = calls.filter((x) => x.path.endsWith("/orders"));
      check(
        JSON.stringify(attempts[0].body) === JSON.stringify(attempts[1].body),
        "429 retry preserves pending confirmation",
      );
      check(
        JSON.stringify(attempts[1].body) === JSON.stringify(attempts[2].body),
        "human verification preserves the pending confirmation",
      );
      const sent = attempts.slice(2);
      check(
        creates === 5 && quotes === 2,
        "expected quote and create attempts",
      );
      check(
        sent[1].headers["idempotency-key"] ===
          sent[2].headers["idempotency-key"],
        "unknown confirmation retried with identical key",
      );
      check(
        JSON.stringify(sent[1].body) === JSON.stringify(sent[2].body),
        "unknown confirmation retried with identical data",
      );
      check(
        sent[1].headers["x-buyer-tenant"] === tenant &&
          !sent[1].headers.authorization,
        "cookie session scoped by restaurant, no secret in JS header",
      );
      check(
        !JSON.stringify(sent[1].body).includes("price"),
        "client cannot set server prices",
      );
      check(
        sent[1].body.fulfillment === "delivery" &&
          sent[1].body.shipping_method_id === shipping,
        "scoped shipping",
      );
      check(
        sent[1].body.items[0].note === "Sin cebolla" &&
          sent[1].body.items[0].extras[0].extra_id === extra,
        "extras and preparation note",
      );
      check(
        (await page.$eval(".buyer-sheet", (el) => el.textContent)).includes(
          "Pendiente de pago",
        ),
        "not falsely marked paid",
      );
      check(
        !(await page.$(".buyer-sheet .buyer-actions .btn-primary")),
        "cannot pay before acceptance",
      );
      const stored = await page.evaluate(() =>
        JSON.stringify({ ...sessionStorage, ...localStorage }),
      );
      check(
        !stored.includes("Cliente de prueba") &&
          !stored.includes("buyer@example.invalid") &&
          !stored.includes("Calle de prueba"),
        "PII not persisted",
      );
      await page.screenshot({
        path: `.cache/buyer-browser/${width}-${theme}-receipt.png`,
      });
      await page.reload({ waitUntil: "networkidle0" });
      await page.waitForSelector(".buyer-receipt-status");
      check(
        calls.filter((x) => x.path.endsWith("/orders")).length === 5,
        "reload recovers receipt without creating order",
      );
      status = "accepted";
      online = true;
      await button(page, "Actualizar estado");
      await page.waitForFunction(() =>
        [...document.querySelectorAll("button")].some((x) =>
          x.textContent.includes("Pagar con Mercado Pago"),
        ),
      );
      check(
        (
          await page.$eval(".buyer-receipt-status", (el) => el.textContent)
        ).includes("Aceptado"),
        "accepted receipt enables real provider checkout",
      );
      order.total_paid = "241.00";
      await button(page, "Actualizar estado");
      await page.waitForFunction(() =>
        document
          .querySelector(".buyer-payment-state")
          ?.textContent.includes("Pago registrado"),
      );
      check(
        !(await page.$(".buyer-sheet .buyer-actions .btn-primary")),
        "paid order cannot be paid again",
      );
      check(
        !(await page.$eval(".buyer-sheet", (el) =>
          [...el.querySelectorAll("button")].some((b) =>
            b.textContent.includes("Armar otro pedido"),
          ),
        )),
        "paid but undelivered order blocks new purchase",
      );
      check(
        (await page.$eval(".buyer-sheet", (el) => el.textContent)).includes(
          "Ya tenés un pedido en curso",
        ),
        "active order explains purchase restriction",
      );
      status = "delivered";
      await button(page, "Actualizar estado");
      await button(page, "Armar otro pedido");
      await page.goto(origin + "/catalogo/" + otherTenant, {
        waitUntil: "networkidle0",
      });
      check(
        !(await page.$(".menu-add")) && !(await page.$(".buyer-cart-trigger")),
        "read-only restaurant and tenant isolation",
      );
      check(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
        "no mobile overflow",
      );
      check(errors.length === 0, JSON.stringify(errors));
      await page.close();
    }
  console.log(
    `Buyer browser: ${assertions} assertions passed (desktop/mobile, light/dark).`,
  );
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
}
