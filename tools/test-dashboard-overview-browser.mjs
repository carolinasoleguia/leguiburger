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
await mkdir(".cache/dashboard-overview", { recursive: true });
let checks = 0,
  currentPage;
async function ready(page) {
  await page.waitForSelector(".stable-content.is-ready");
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
}
async function click(page, text) {
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
    for (const theme of ["light", "dark"])
      for (const role of ["admin", "owner"]) {
        const page = (currentPage = await browser.newPage()),
          errors = [],
          calls = [];
        let failNext = false;
        await page.setViewport({ width, height: width > 800 ? 768 : 900 });
        await page.evaluateOnNewDocument((theme) => {
          localStorage.setItem("legui-theme", theme);
          const NativeDate = Date;
          // UTC March 1 is still February 28 in Argentina: presets cross months.
          window.Date = class extends NativeDate {
            constructor(...args) {
              super(...(args.length ? args : ["2026-03-01T02:30:00Z"]));
            }
            static now() {
              return new NativeDate("2026-03-01T02:30:00Z").getTime();
            }
          };
        }, theme);
        page.on("pageerror", (e) => errors.push(e.message));
        page.on("dialog", (d) => d.accept());
        await page.setRequestInterception(true);
        page.on("request", async (request) => {
          const url = new URL(request.url()),
            path = url.pathname;
          if (!path.startsWith("/api/")) return request.continue();
          calls.push({
            path,
            query: Object.fromEntries(url.searchParams),
            headers: request.headers(),
          });
          let data = { items: [] },
            status = 200;
          if (path === "/api/auth/refresh")
            data = {
              token: "fixture",
              user: {
                id: "fixture-user",
                role,
                brand_id: role === "admin" ? "brand-1" : undefined,
                email: "fixture@example.invalid",
              },
            };
          else if (path === "/api/tenants")
            data = [{ id: "tenant-1", subdomain: "Centro", active: true }];
          else if (path.startsWith("/api/analytics/")) {
            if (failNext) {
              failNext = false;
              status = 503;
              data = { message: "Consulta temporalmente no disponible" };
            } else {
              const from = url.searchParams.get("from"),
                to = url.searchParams.get("to"),
                count =
                  Math.round((Date.parse(to) - Date.parse(from)) / 86400000) +
                  1;
              if (path.endsWith("/restaurant"))
                data = {
                  metrics: {
                    orders_created: count,
                    new_customers: 2,
                    returning_customers: 3,
                  },
                  money: [
                    {
                      currency: "ARS",
                      sales: String(count * 1000),
                      collected: String(count * 900),
                      average_ticket: "1000",
                      operating_expenses: String(count * 100),
                      estimated_operating_result: null,
                      missing_cost_lines: 1,
                    },
                  ],
                  previous_period: {
                    sales: [{ currency: "ARS", amount: String(count * 500) }],
                  },
                  daily_sales: [
                    { day: from, currency: "ARS", total: 1000 },
                    { day: to, currency: "ARS", total: 2000 },
                  ],
                  order_states: [
                    { status: "pending", count: 2 },
                    { status: "preparing", count: 3 },
                    { status: "delivered", count: 5 },
                  ],
                  top_products: [
                    {
                      product_id: "product-1",
                      product_name: "Doble cheddar",
                      currency: "ARS",
                      units: 5,
                    },
                  ],
                  recent_orders: [
                    {
                      id: "order-fixture-1",
                      status: "delivered",
                      total_due: "1000",
                      currency: "ARS",
                      created_at: "2026-02-28T18:00:00Z",
                    },
                  ],
                };
              else
                data = {
                  money: [
                    {
                      currency: "ARS",
                      collected: "50000",
                      active_mrr: "100000",
                      outstanding: "5000",
                    },
                  ],
                  businesses: {
                    production_registered_in_period: 3,
                    active_last_week: 6,
                    unclassified: 1,
                  },
                  subscriptions: {
                    active: 6,
                    trialing: 2,
                    past_due: 1,
                    canceled_in_period: 1,
                  },
                  daily_new_businesses: [
                    { day: from, registered: 1 },
                    { day: to, registered: 2 },
                  ],
                  activity_by_feature: [],
                };
            }
          }
          await request.respond({
            status,
            contentType: "application/json",
            body: JSON.stringify(data),
          });
        });
        await page.goto(origin + (role === "admin" ? "/admin" : "/owner"), {
          waitUntil: "domcontentloaded",
        });
        await ready(page);
        if (width > 800) {
          assert.ok(
            await page.$eval(".sidebar nav", (nav) =>
              [...nav.querySelectorAll(".nav-group-toggle")].every(
                (button) =>
                  button.getBoundingClientRect().bottom <=
                  nav.getBoundingClientRect().bottom + 1,
              ),
            ),
            "all menu group headings fit in a notebook sidebar",
          );
          checks++;
        }
        const analytics = () =>
          calls.filter((c) => c.path.startsWith("/api/analytics/")).at(-1);
        assert.equal(
          analytics().path,
          role === "admin"
            ? "/api/analytics/restaurant"
            : "/api/analytics/platform",
        );
        assert.deepEqual(analytics().query, {
          from: "2026-02-22",
          to: "2026-02-28",
        });
        assert.equal(await page.$("main input[type=date]"), null);
        if (role === "admin") {
          assert.equal(
            await page.$eval("main", (e) =>
              /membres[ií]as|suscripciones|ingreso mensual contratado/i.test(
                e.textContent,
              ),
            ),
            false,
          );
          assert.equal(
            calls.some((c) => c.path === "/api/analytics/platform"),
            false,
          );
          assert.equal(analytics().headers["x-tenant-id"], "tenant-1");
        } else
          assert.equal(
            await page.$eval("main", (e) =>
              e.textContent.includes("Cobros de membresías"),
            ),
            true,
          );
        checks += 5;
        await click(page, "15 días");
        await ready(page);
        assert.deepEqual(analytics().query, {
          from: "2026-02-14",
          to: "2026-02-28",
        });
        await click(page, "1 mes");
        await ready(page);
        assert.deepEqual(analytics().query, {
          from: "2026-01-30",
          to: "2026-02-28",
        });
        assert.equal(
          await page.$eval(".overview-range-presets [aria-pressed=true]", (e) =>
            e.textContent.trim(),
          ),
          "1 mes",
        );
        const path = await page.$eval(".chart-line", (e) =>
          e.getAttribute("d"),
        );
        assert.ok(path.includes("C") && !/NaN|Infinity/.test(path));
        checks += 4;
        await page.click(".overview-area-chart");
        await page.focus(".overview-area-chart");
        await page.keyboard.press("Home");
        await page.waitForSelector(".chart-tooltip");
        assert.equal(
          await page.$eval(".chart-tooltip", (e) =>
            e.textContent.includes("30 de enero"),
          ),
          true,
        );
        await page.keyboard.press("End");
        assert.equal(
          await page.$eval(".chart-tooltip", (e) =>
            e.textContent.includes("28 de febrero"),
          ),
          true,
        );
        await page.click(".overview-calendar-trigger");
        await page.waitForSelector(".calendar-grid");
        await page.click('[data-date="2026-02-10"]');
        await page.click('[data-date="2026-02-18"]');
        await click(page, "Aplicar período");
        await ready(page);
        assert.deepEqual(analytics().query, {
          from: "2026-02-10",
          to: "2026-02-18",
        });
        assert.equal(
          await page.$(".overview-range-presets [aria-pressed=true]"),
          null,
        );
        await page.click(".overview-calendar-trigger");
        await page.keyboard.press("Escape");
        await page.waitForSelector(".calendar-grid", { hidden: true });
        assert.equal(await page.$(".calendar-grid"), null);
        checks += 5;
        if (role === "admin" && width === 1366 && theme === "light") {
          failNext = true;
          await click(page, "7 días");
          await page.waitForSelector("main [role=alert]");
          await click(page, "Reintentar");
          await ready(page);
          assert.deepEqual(analytics().query, {
            from: "2026-02-22",
            to: "2026-02-28",
          });
          checks++;
        }
        if (width < 800) {
          await page.click('button[aria-label="Abrir menú"]');
          await page.waitForFunction(() => {
            const panel = document.querySelector(".sidebar.is-open");
            return (
              panel &&
              panel.getBoundingClientRect().left >= -0.5 &&
              getComputedStyle(panel).visibility === "visible"
            );
          });
        }
        await page.click('.nav-group-toggle[aria-controls$="-menu"]');
        await page.waitForSelector(
          '.nav-group-items a[href="/product-options"]',
          { visible: true },
        );
        await page.waitForFunction(
          () =>
            !document.querySelector(
              ".nav-group-enter-active, .nav-group-leave-active",
            ),
        );
        const nav = await page.$eval(".sidebar nav", (e) => ({
          count: e.querySelectorAll("a").length,
          buttons: e.querySelectorAll(".nav-group-toggle").length,
        }));
        assert.ok(nav.count < 18 && nav.buttons >= 4);
        if (role === "admin") {
          await page.click('.nav-group-items a[href="/product-options"]');
          await page.waitForFunction(
            () => location.pathname === "/product-options",
          );
          assert.equal(
            await page.$eval('button[aria-controls$="-menu"]', (e) =>
              e.getAttribute("aria-expanded"),
            ),
            "true",
          );
        }
        checks += 2;
        await page.goto(origin + (role === "admin" ? "/admin" : "/owner"), {
          waitUntil: "domcontentloaded",
        });
        await ready(page);
        assert.equal(
          await page.evaluate(
            () => document.documentElement.scrollWidth > innerWidth + 1,
          ),
          false,
        );
        if (width > 800)
          assert.ok(
            await page.$eval(
              ".overview-activity",
              (e) => e.getBoundingClientRect().bottom <= innerHeight,
            ),
            "main chart and states fit in notebook viewport",
          );
        await page.screenshot({
          path: `.cache/dashboard-overview/${role}-${width}-${theme}.png`,
          fullPage: true,
        });
        assert.deepEqual(errors, []);
        checks += 3;
        await page.close();
      }
  console.log(
    `Dashboard overview: ${checks} checks passed: separate restaurant/platform queries, membership isolation, 7/15/30-day periods, Argentina month boundary, single range calendar, SVG area/tooltip/keyboard, retry and grouped navigation at notebook/mobile in light/dark.`,
  );
} catch (error) {
  if (currentPage && !currentPage.isClosed()) {
    console.error(
      currentPage.url(),
      await currentPage.$eval("body", (e) => e.innerText.slice(-3000)),
    );
    await currentPage.screenshot({
      path: ".cache/dashboard-overview/failure.png",
      fullPage: true,
    });
  }
  throw error;
} finally {
  await browser.close();
  server.closeAllConnections();
  await new Promise((r) => server.close(r));
}
