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
  blocked = true,
  resolved = false,
  fail = true;
const actions = [],
  errors = [];
const server = createServer(async (req, res) => {
  const url = new URL(req.url, "http://localhost"),
    path = url.pathname;
  const reply = (data, status = 200) => {
    res.writeHead(status, { "Content-Type": "application/json" });
    res.end(JSON.stringify(data));
  };
  if (path.startsWith("/api/")) {
    let raw = "";
    for await (const chunk of req) raw += chunk;
    if (path === "/api/auth/refresh")
      return reply({
        token: "fixture",
        user: {
          id: "fixture-owner",
          role,
          brand_id: "brand",
          tenant_id: role === "cashier" ? "tenant" : undefined,
        },
      });
    if (path === "/api/notifications") return reply({ items: [] });
    if (path === "/api/platform/security/incidents") {
      const item = {
        id: "incident-1",
        label: "198.51.100.0/24",
        reference: "4d7a95b22a1e",
        reason: "session_churn",
        occurrences: 14,
        evidence: { new_sessions: 14, invalid_requests: 3, orders: 2 },
        business: "Burger de barrio",
        location: "Centro",
        first_seen_at: new Date().toISOString(),
        last_seen_at: new Date().toISOString(),
        blocked_until: blocked
          ? new Date(Date.now() + 900000).toISOString()
          : null,
        resolved_at: resolved ? new Date().toISOString() : null,
      };
      const items =
        (url.searchParams.get("status") === "open" && resolved) ||
        (url.searchParams.get("status") === "blocked" && !blocked)
          ? []
          : [item];
      return reply({
        items,
        total: items.length,
        pending: resolved ? 0 : 1,
        blocked: blocked ? 1 : 0,
        verification_configured: false,
      });
    }
    if (path.endsWith("/actions")) {
      const body = JSON.parse(raw);
      actions.push({ body, key: req.headers["idempotency-key"] });
      if (fail) {
        fail = false;
        return reply(
          { message: "Respuesta demorada, reintentá la misma acción" },
          503,
        );
      }
      if (body.action === "unblock") {
        blocked = false;
        resolved = true;
      }
      if (body.action === "block") {
        blocked = true;
        resolved = false;
      }
      if (body.action === "resolve") resolved = true;
      return reply({ ok: true });
    }
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
await mkdir(".cache/checkout-security", { recursive: true });
let page;
async function click(text) {
  await page.waitForFunction(
    (t) =>
      [...document.querySelectorAll("button")].some(
        (b) => b.textContent.trim() === t && !b.disabled,
      ),
    {},
    text,
  );
  await page.evaluate(
    (t) =>
      [...document.querySelectorAll("button")]
        .find((b) => b.textContent.trim() === t && !b.disabled)
        .click(),
    text,
  );
}
try {
  for (const width of [1366, 390])
    for (const theme of ["light", "dark"]) {
      role = "owner";
      blocked = true;
      resolved = false;
      fail = true;
      actions.length = 0;
      page = await browser.newPage();
      page.on("pageerror", (e) => errors.push(e.message));
      await page.setViewport({ width, height: 900 });
      await page.evaluateOnNewDocument(
        (t) => localStorage.setItem("legui-theme", t),
        theme,
      );
      await page.goto(origin + "/platform/security", {
        waitUntil: "networkidle0",
      });
      await page.waitForSelector(".stable-content.is-ready");
      assert(
        await page.$eval("main", (e) =>
          e.textContent.includes("Sin configurar"),
        ),
      );
      assert(
        await page.$eval("main", (e) =>
          e.textContent.includes("Muchas sesiones nuevas"),
        ),
      );
      await click("Desbloquear");
      assert.equal(
        await page.$eval(".incident-action .btn-primary", (e) => e.disabled),
        true,
      );
      await page.type(
        ".incident-action textarea",
        "Verificamos que era una conexión compartida.",
      );
      await click("Guardar acción");
      await page.waitForSelector("[role=alert]");
      assert.equal(
        await page.$eval(".incident-action textarea", (e) => e.value),
        "Verificamos que era una conexión compartida.",
      );
      await click("Guardar acción");
      await page.waitForSelector("[role=status]");
      await page.waitForSelector(".incident-action", { hidden: true });
      assert.equal(actions[0].key, actions[1].key);
      assert.deepEqual(actions[0].body, actions[1].body);
      assert.equal(actions[1].body.action, "unblock");
      await page.select(".workspace-filters select", "all");
      await page.waitForFunction(() =>
        document.querySelector("tbody")?.textContent.includes("Resuelto"),
      );
      await click("Bloquear");
      await page.select(".incident-action select", "60");
      await page.type(
        ".incident-action textarea",
        "Revisamos nuevos intentos abusivos persistentes.",
      );
      await click("Guardar acción");
      await page.waitForSelector(".incident-action", { hidden: true });
      assert.equal(actions.at(-1).body.minutes, 60);
      assert.equal(
        await page.evaluate(
          () => document.documentElement.scrollWidth > innerWidth + 1,
        ),
        false,
      );
      await page.waitForFunction(() =>
        document.querySelector('.stable-content.is-ready[aria-busy="false"]'),
      );
      await page.evaluate(() => document.fonts.ready);
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
        path: `.cache/checkout-security/owner-${width}-${theme}.png`,
        fullPage: true,
      });
      for (const denied of ["admin", "cashier"]) {
        role = denied;
        await page.goto(origin + "/platform/security", {
          waitUntil: "networkidle0",
        });
        assert.equal(new URL(page.url()).pathname, "/unauthorized");
      }
      await page.close();
    }
  assert.deepEqual(errors, []);
  console.log(
    "Checkout security browser passed: owner-only panel, filters, evidence, missing configuration, reason validation, retry idempotency and temporary block/unblock at notebook/mobile in light/dark.",
  );
} catch (e) {
  if (page && !page.isClosed()) {
    console.error(
      page.url(),
      await page.$eval("body", (e) => e.innerText.slice(-2400)),
    );
    await page.screenshot({
      path: ".cache/checkout-security/failure.png",
      fullPage: true,
    });
  }
  throw e;
} finally {
  await browser.close();
  server.closeAllConnections();
  await new Promise((r) => server.close(r));
}
