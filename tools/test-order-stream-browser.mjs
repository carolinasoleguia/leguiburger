import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { resolve, sep, extname } from "node:path";
import { pathToFileURL } from "node:url";
const { default: puppeteer } = await import(
  pathToFileURL(process.env.PUPPETEER_MODULE).href
);
const dist = resolve("frontend/dist"),
  token = "t".repeat(43),
  clients = new Set();
let status = "pending",
  paid = "0.00",
  allowStream = true,
  reads = 0,
  opened = 0;
const server = createServer(async (req, res) => {
  const url = new URL(req.url, "http://localhost");
  if (url.pathname.startsWith("/api/")) {
    if (url.pathname.startsWith("/api/auth/")) {
      res.writeHead(401, { "Content-Type": "application/json" });
      res.end("{}");
      return;
    }
    if (req.headers.authorization !== "Bearer " + token) {
      res.writeHead(404);
      res.end();
      return;
    }
    assert.equal(url.search, "", "capability must not enter URL");
    if (url.pathname === "/api/public/order/events") {
      opened++;
      if (!allowStream) {
        res.writeHead(503);
        res.end();
        return;
      }
      res.writeHead(200, {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-store",
      });
      res.write("event: ready\ndata: {}\n\n");
      clients.add(res);
      req.on("close", () => clients.delete(res));
      return;
    }
    if (url.pathname === "/api/public/order") {
      reads++;
      res.writeHead(200, { "Content-Type": "application/json" });
      res.end(
        JSON.stringify({
          id: "fixture",
          status,
          total_due: "100.00",
          total_paid: paid,
          business: "Burger de barrio",
          currency: "ARS",
          created_at: new Date().toISOString(),
          items: [],
        }),
      );
      return;
    }
    res.writeHead(404);
    res.end();
    return;
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
const browser = await puppeteer.launch({
  headless: true,
  executablePath: process.env.CHROME_PATH,
  args: process.env.CI ? ["--no-sandbox"] : [],
});
const event = () => {
  for (const client of clients) client.write("event: changed\ndata: {}\n\n");
};
try {
  const page = await browser.newPage();
  const errors = [];
  page.on("pageerror", (e) => errors.push(e.message));
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
  await page.goto(
    `http://127.0.0.1:${server.address().port}/order#token=${token}`,
    { waitUntil: "domcontentloaded" },
  );
  await page.waitForFunction(() =>
    document.body.textContent.includes("Seguimiento en vivo"),
  );
  assert.equal(
    await page.evaluate(() => window.chimes),
    0,
    "initial snapshot silent",
  );
  await page.evaluate(() =>
    [...document.querySelectorAll("button")]
      .find((b) => b.textContent.includes("Activar sonido"))
      .click(),
  );
  await page.waitForFunction(() =>
    document.querySelector("[aria-pressed=true]"),
  );
  status = "accepted";
  event();
  await page.waitForFunction(
    () => document.body.textContent.includes("Aceptado") && window.chimes === 2,
  );
  event();
  await new Promise((r) => setTimeout(r, 150));
  assert.equal(
    await page.evaluate(() => window.chimes),
    2,
    "duplicate event silent",
  );
  // Reconnection must resynchronize state even without a delivered notification.
  status = "preparing";
  for (const c of clients) c.end();
  await page.waitForFunction(
    () =>
      document.body.textContent.includes("En preparación") &&
      window.chimes === 4,
  );
  paid = "100.00";
  event();
  await page.waitForFunction(() => window.chimes === 6);
  await page.evaluate(() =>
    [...document.querySelectorAll("button")]
      .find((b) => b.textContent.includes("Silenciar avisos"))
      .click(),
  );
  status = "ready";
  event();
  await page.waitForFunction(() => document.body.textContent.includes("Listo"));
  assert.equal(await page.evaluate(() => window.chimes), 6, "muted sound");
  allowStream = false;
  for (const c of clients) c.end();
  status = "dispatched";
  await page.evaluate(() => {
    const original = Date.now;
    Date.now = () => original() + 31000;
  });
  await page.waitForFunction(
    () => document.body.textContent.includes("En camino"),
    { timeout: 15000 },
  );
  assert.ok(reads < 20, "bounded reads");
  assert.ok(opened >= 2, "reconnected");
  assert.deepEqual(errors, []);
  console.log(
    "SSE browser passed: header authorization, initial sync, committed-change refresh, reconnect, polling fallback, opt-in sound, duplicate suppression and mute.",
  );
} finally {
  for (const c of clients) c.end();
  await browser.close();
  await new Promise((r) => server.close(r));
}
