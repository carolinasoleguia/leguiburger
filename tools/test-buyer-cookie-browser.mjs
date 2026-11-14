import { paidMembership } from "./membership-fixture.mjs";
import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { resolve, sep, extname } from "node:path";
import { pathToFileURL } from "node:url";
import { randomUUID } from "node:crypto";
const { default: puppeteer } = await import(
  pathToFileURL(process.env.PUPPETEER_MODULE).href
);
const dist = resolve("frontend/dist"),
  tenant = "10000000-0000-4000-8000-000000000001",
  other = "10000000-0000-4000-8000-000000000002";
const sessions = new Map();
let status = "pending";
const cookieName = (id) => "legui_buyer_" + id.replaceAll("-", "");
const server = createServer(async (req, res) => {
  if (new URL(req.url, 'http://localhost').pathname === '/api/membership') { res.writeHead(200, {'Content-Type':'application/json'});res.end(JSON.stringify(paidMembership));return; }
  const url = new URL(req.url, "http://localhost");
  const reply = (data, code = 200) => {
    res.writeHead(code, {
      "Content-Type": "application/json",
      "Cache-Control": "no-store",
    });
    res.end(JSON.stringify(data));
  };
  if (url.pathname.startsWith("/api/")) {
    if (url.pathname.endsWith("/events")) return reply({}, 503);
    if (url.pathname.startsWith("/api/auth/")) return reply({}, 401);
    const id = url.pathname.startsWith("/api/public/menu/")
      ? url.pathname.split("/")[4]
      : req.headers["x-buyer-tenant"];
    let secret = (req.headers.cookie || "")
      .split("; ")
      .find((x) => x.startsWith(cookieName(id || "") + "="))
      ?.split("=")[1];
    if (url.pathname.endsWith("/buyer-session")) {
      let body = "";
      for await (const chunk of req) body += chunk;
      if (!sessions.has(secret) || JSON.parse(body).new) {
        secret = randomUUID();
        sessions.set(secret, {
          key: randomUUID(),
          paymentKey: randomUUID(),
          order: null,
        });
        res.setHeader(
          "Set-Cookie",
          `${cookieName(id)}=${secret}; Path=/api/public; HttpOnly; SameSite=Lax; Max-Age=86400`,
        );
      }
      const session = sessions.get(secret);
      return reply({
        creation_key: session.key,
        payment_key: session.paymentKey,
      });
    }
    if (url.pathname === "/api/public/buyer-order") {
      const order = sessions.get(secret)?.order;
      return order ? reply({ ...order, status }) : reply({}, 404);
    }
    if (url.pathname.startsWith("/api/public/menu/"))
      return reply({
        location: { id, name: "City Bell", brand_name: "Burger de barrio" },
        ordering_enabled: true,
        currency: "ARS",
        total: 1,
        categories: ["Hamburguesas"],
        extras: [],
        shipping_methods: [],
        items: [
          {
            id: randomUUID(),
            name: "Doble de la casa",
            description: "Carne, cheddar y salsa",
            price: "100.00",
            available: true,
            category: "Hamburguesas",
            image_url: "",
          },
        ],
      });
    return reply({}, 404);
  }
  let file = resolve(dist, "." + url.pathname);
  if (!file.startsWith(dist + sep) || !extname(file))
    file = resolve(dist, "index.html");
  try {
    res.setHeader(
      "Content-Type",
      {
        ".html": "text/html",
        ".js": "application/javascript",
        ".css": "text/css",
        ".svg": "image/svg+xml",
        ".woff2": "font/woff2",
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
try {
  const context = await browser.createBrowserContext();
  let page = await context.newPage();
  await page.goto(origin + "/catalogo/" + tenant, {
    waitUntil: "domcontentloaded",
  });
  await page.waitForSelector(".menu-add");
  await page.waitForNetworkIdle({idleTime:100,timeout:5000});
  const cookie = (await context.cookies()).find(
    (c) => c.name === cookieName(tenant),
  );
  assert.ok(cookie?.httpOnly);
  assert.ok(cookie.expires > Date.now() / 1000);
  assert.equal(cookie.path, "/api/public");
  const saved = sessions.get(cookie.value);
  assert.ok(saved);
  saved.order = {
    id: "50000000-0000-4000-8000-000000000001",
    tenant_id: tenant,
    total_due: "100.00",
    total_paid: "0.00",
    fulfillment: "pickup",
    payment_method: "at_local",
    business: "Burger de barrio",
    location: "City Bell",
    online_available: false,
    items: [],
  };
  await page.evaluate(() => {
    sessionStorage.clear();
    localStorage.clear();
  });
  await page.close();
  page = await context.newPage();
  await page.goto(origin + "/catalogo/" + tenant, {
    waitUntil: "domcontentloaded",
  });
  await page.waitForSelector(".buyer-receipt-status");
  assert.ok(
    (await page.$eval(".buyer-sheet", (el) => el.textContent)).includes(
      "Ya tenés un pedido en curso",
    ),
  );
  assert.ok(
    !(await page.evaluate(() => document.cookie.includes("legui_buyer_"))),
    "HttpOnly not readable",
  );
  assert.equal(
    await page.evaluate(() => sessionStorage.length),
    0,
    "no secret in tab storage",
  );
  await page.close();
  status = "delivered";
  saved.order.total_paid = "100.00";
  page = await context.newPage();
  await page.goto(origin + "/catalogo/" + tenant, {
    waitUntil: "domcontentloaded",
  });
  await page.waitForSelector(".buyer-receipt-status");
  assert.ok(
    (await page.$eval(".buyer-sheet", (el) => el.textContent)).includes(
      "Entregado",
    ),
    "finished order still recoverable",
  );
  await page.goto(origin + "/catalogo/" + other, { waitUntil: "domcontentloaded" });
  await page.waitForSelector(".menu-add");
  await page.waitForNetworkIdle({idleTime:100,timeout:5000});
  assert.equal(
    await page.$(".buyer-receipt-status"),
    null,
    "another restaurant has separate cookie",
  );
  const fresh = await browser.createBrowserContext();
  const freshPage = await fresh.newPage();
  await freshPage.goto(origin + "/catalogo/" + tenant, {
    waitUntil: "domcontentloaded",
  });
  await freshPage.waitForSelector(".menu-add");
  await freshPage.waitForNetworkIdle({idleTime:100,timeout:5000});
  assert.equal(
    await freshPage.$(".buyer-receipt-status"),
    null,
    "different browser never inherits order",
  );
  await fresh.close();
  await context.close();
  console.log(
    "Buyer cookie browser passed: HttpOnly persistence, fresh tab recovery without storage, completed receipt retained, restaurant isolation and separate browser isolation.",
  );
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
}
