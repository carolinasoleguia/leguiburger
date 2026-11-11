import assert from "node:assert/strict";
import { createServer } from "node:http";
import { readFile, mkdir } from "node:fs/promises";
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
    const content = await readFile(file);
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
    res.end(content);
  } catch {
    res.writeHead(404);
    res.end();
  }
});
await new Promise((resolve) => server.listen(0, "127.0.0.1", resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
const browser = await puppeteer.launch({
  headless: true,
  executablePath: process.env.CHROME_PATH,
  args: ["--disable-gpu", ...(process.env.CI ? ["--no-sandbox"] : [])],
});
await mkdir(".cache/motion-browser", { recursive: true });
const queued = [],
  errors = [],
  page = await browser.newPage();
let hold = true,
  noTenants = false,
  productionRequests = 0;
const locals = {
    items: [
      {
        id: "tenant-1",
        brand_id: "brand-1",
        subdomain: "centro",
        display_name: "Centro",
        active: true,
        address: "",
        contact_phone: "",
        service_hours: "",
      },
    ],
  },
  business = { items: [{ id: "brand-1", name: "Burger de barrio" }] };
const respond = (request, data, status = 200) =>
  request.respond({
    status,
    contentType: "application/json",
    body: JSON.stringify(data),
  });
async function waitQueued() {
  const started = Date.now();
  while (queued.length < 2) {
    if (Date.now() - started > 10000)
      throw new Error("Expected two independent data requests");
    await new Promise((resolve) => setTimeout(resolve, 20));
  }
}
async function release(status = 200) {
  const requests = queued.splice(0);
  for (const req of requests)
    await respond(
      req,
      new URL(req.url()).pathname === "/api/business/locations"
        ? status === 200
          ? locals
          : {
              message:
                "No pudimos actualizar el local. Tu borrador sigue disponible.",
            }
        : business,
      new URL(req.url()).pathname === "/api/business/locations" ? status : 200,
    );
}
try {
  page.on("pageerror", (e) => errors.push(e.message));
  await page.emulateMediaFeatures([
    { name: "prefers-reduced-motion", value: "no-preference" },
  ]);
  page.on("dialog", (dialog) => dialog.accept());
  await page.setViewport({ width: 1440, height: 1000 });
  await page.setRequestInterception(true);
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (!path.startsWith("/api/")) return request.continue();
    if (path === "/api/auth/refresh")
      return respond(request, {
        token: "fixture-token",
        user: {
          id: "owner-1",
          brand_id: "brand-1",
          role: "admin",
          email: "owner@example.invalid",
        },
      });
    if (path === "/api/production") {
      productionRequests++;
      return respond(request, []);
    }
    if (path === "/api/tenants" && noTenants) return respond(request, []);
    if (path === "/api/tenants")
      return respond(request, [{ ...locals.items[0], name: "Centro" }]);
    if (path === "/api/business/locations" || path === "/api/business") {
      if (hold) {
        queued.push(request);
        return;
      }
      return respond(request, path.endsWith("locations") ? locals : business);
    }
    return respond(request, []);
  });
  await page.goto(origin + "/locations", { waitUntil: "domcontentloaded" });
  await page.waitForSelector("[data-dashboard-loading]", { visible: true });
  await waitQueued();
  assert.equal(
    await page.$eval("[data-dashboard-loading]", (el) =>
      el.getAttribute("aria-busy"),
    ),
    "true",
  );
  assert(
    await page.$eval(".skeleton-bone", (el) =>
      getComputedStyle(el).animationName.includes("skeleton-pulse"),
    ),
  );
  await page.waitForSelector(".app-progress", { visible: true });
  await page.screenshot({
    path: ".cache/motion-browser/skeleton-1440-dark.png",
    fullPage: true,
  });
  const one = queued.pop();
  await respond(
    one,
    new URL(one.url()).pathname.endsWith("locations") ? locals : business,
  );
  assert(
    await page.$(".app-progress"),
    "activity must remain while another request is pending",
  );
  await page.emulateMediaFeatures([
    { name: "prefers-reduced-motion", value: "reduce" },
  ]);
  assert.equal(
    await page.$eval(
      ".skeleton-bone",
      (el) => getComputedStyle(el).animationName,
    ),
    "none",
  );
  await page.setViewport({ width: 390, height: 950 });
  await page.evaluate(() => (document.documentElement.dataset.theme = "light"));
  assert(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth + 1,
    ),
  );
  await page.screenshot({
    path: ".cache/motion-browser/skeleton-390-light.png",
    fullPage: true,
  });
  await page.emulateMediaFeatures([
    { name: "prefers-reduced-motion", value: "no-preference" },
  ]);
  await page.setViewport({ width: 1440, height: 1000 });
  await page.evaluate(() => {
    document.documentElement.dataset.theme = "dark";
    window.__panelTransitions = 0;
    document.addEventListener("transitionrun", (event) => {
      if (event.target.classList.contains("data-stage-content"))
        window.__panelTransitions++;
    });
  });
  await release();
  await page.waitForSelector("[data-dashboard-loading]", { hidden: true });
  await page.waitForFunction(() => window.__panelTransitions > 0);
  await page.waitForSelector(".app-progress", { hidden: true });
  await page.evaluate(() => {
    const label = [...document.querySelectorAll("label")].find((el) =>
      el.textContent.includes("Identificador"),
    );
    const input = label.querySelector("input");
    input.value = "mi-borrador";
    input.dispatchEvent(new Event("input", { bubbles: true }));
  });
  await page.evaluate(() =>
    [...document.querySelectorAll("button")]
      .find((el) => el.textContent.includes("Actualizar"))
      .click(),
  );
  await waitQueued();
  await page.waitForSelector('button.is-pending[aria-busy="true"]');
  assert.equal(
    await page.$eval(".data-stage-content input", (el) => el.value),
    "mi-borrador",
  );
  await release(503);
  await page.waitForSelector('[role="alert"]');
  await page.waitForSelector("button.is-pending", { hidden: true });
  assert.equal(
    await page.$eval(".data-stage-content input", (el) => el.value),
    "mi-borrador",
  );
  hold = false;
  await page.evaluate(() =>
    [...document.querySelectorAll('a[href="/admin/catalog"]')]
      .find((el) => el.getClientRects().length)
      .click(),
  );
  await page.waitForSelector(".confirmation-card");
  await page.waitForFunction(() =>
    document
      .querySelector(".confirmation-card")
      ?.contains(document.activeElement),
  );
  assert.equal(new URL(page.url()).pathname, "/locations");
  await page.keyboard.press("Escape");
  await page.waitForSelector(".confirmation-card", { hidden: true });
  assert.equal(new URL(page.url()).pathname, "/locations");
  await page.evaluate(() => {
    const input = document.querySelector(".data-stage-content input");
    input.value = "";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    [...document.querySelectorAll('a[href="/admin/catalog"]')]
      .find((el) => el.getClientRects().length)
      .click();
  });
  await page.waitForFunction(() =>
    document.querySelector("main h1")?.textContent.includes("catálogo"),
  );
  await page.waitForFunction(() =>
    [...document.querySelectorAll("button")].some((el) =>
      el.textContent.includes("Nuevo producto"),
    ),
  );
  await page.evaluate(() =>
    [...document.querySelectorAll("button")]
      .find((el) => el.textContent.includes("Nuevo producto"))
      .click(),
  );
  await page.waitForSelector('[role="dialog"]', { visible: true });
  assert(
    await page.$eval('[role="dialog"]', (el) =>
      getComputedStyle(el)
        .transitionDuration.split(",")
        .some((value) => parseFloat(value) > 0),
    ),
  );
  await page.emulateMediaFeatures([
    { name: "prefers-reduced-motion", value: "reduce" },
  ]);
  assert(
    await page.$eval('[role="dialog"]', (el) =>
      getComputedStyle(el)
        .transitionDuration.split(",")
        .every((value) => parseFloat(value) === 0),
    ),
  );
  await page.keyboard.press("Escape");
  await page.waitForSelector('[role="dialog"]', { hidden: true });
  await page.click(".motion-toggle");
  await page.waitForFunction(
    () => document.documentElement.dataset.motion === "full",
  );
  await page.evaluate(() =>
    [...document.querySelectorAll("button")]
      .find((el) => el.textContent.includes("Nuevo producto"))
      .click(),
  );
  await page.waitForSelector('[role="dialog"]', { visible: true });
  assert(
    await page.$eval('[role="dialog"]', (el) =>
      getComputedStyle(el)
        .transitionDuration.split(",")
        .some((value) => parseFloat(value) > 0.1),
    ),
    "explicit motion preference must override system reduction",
  );
  await page.waitForFunction(() =>
    document.querySelector('[role="dialog"]')?.contains(document.activeElement),
  );
  await page.keyboard.press("Escape");
  await page.waitForSelector('[role="dialog"]', { hidden: true });
  await page.click(".motion-toggle");
  await page.waitForFunction(
    () => document.documentElement.dataset.motion === "reduced",
  );
  await page.reload({ waitUntil: "networkidle0" });
  assert.equal(
    await page.evaluate(() => document.documentElement.dataset.motion),
    "reduced",
  );
  noTenants = true;
  await page.goto(origin + "/admin/production", { waitUntil: "networkidle0" });
  await page.waitForSelector("[data-dashboard-loading]", { hidden: true });
  assert(
    await page.$eval("main", (el) =>
      el.textContent.includes("Seleccioná un local para ver los reportes"),
    ),
  );
  assert.equal(productionRequests, 0);
  noTenants = false;
  await page.reload({ waitUntil: "networkidle0" });
  await page.waitForSelector("[data-dashboard-loading]", { hidden: true });
  assert.equal(
    productionRequests,
    1,
    "initial production must not issue duplicate requests",
  );
  assert.deepEqual(errors, []);
  console.log(
    "Motion browser passed: real delayed loading skeletons, theme/mobile bounds, shared activity across concurrent requests, reveal transitions, preserved failed drafts, busy indicators, modal navigation and reduced motion.",
  );
} catch (error) {
  console.error("Motion browser failure", page.url(), errors);
  console.error(await page.evaluate(() => document.body.innerText));
  await page.screenshot({
    path: ".cache/motion-browser/failure.png",
    fullPage: true,
  });
  throw error;
} finally {
  await browser.close();
  await new Promise((resolve) => server.close(resolve));
}
