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
await mkdir(".cache/menu-cash-browser", { recursive: true });

const tenant = "10000000-0000-4000-8000-000000000001",
  product = "20000000-0000-4000-8000-000000000001";
const location = {
  id: tenant,
  subdomain: "centro",
  active: true,
  brand: { id: "brand-1", name: "Burger de barrio" },
};
const menuItems = [
  {
    id: product,
    name: "Doble de la casa",
    description: "Doble carne, cheddar y salsa de la casa",
    category: "Hamburguesas",
    price: "9800.00",
    available: true,
    image_url: "",
  },
  {
    id: "product-2",
    name: "Papas con cheddar",
    description: "Papas recién hechas",
    category: "Acompañamientos",
    price: "4200.00",
    available: false,
    image_url: "",
  },
];
let currentPage;
async function setup(width, theme, actor = "admin") {
  const page = await browser.newPage();
  currentPage = page;
  const errors = [],
    calls = [];
  let published = false,
    shift = null,
    entries = [],
    holdMenu = false,
    pendingMenu;
  page.on("pageerror", (e) => errors.push(e.message));
  await page.setViewport({ width, height: 950 });
  await page.evaluateOnNewDocument(
    (theme) => localStorage.setItem("legui-theme", theme),
    theme,
  );
  await page.setRequestInterception(true);
  const reply = (r, data, status = 200) =>
    r.respond({
      status,
      contentType: "application/json",
      body: JSON.stringify(data),
    });
  function snapshot() {
    return {
      ...shift,
      opening_balance: "100.00",
      cash_charged: "0.00",
      cash_refunded: "0.00",
      cash_in: entries.length ? "20.00" : "0.00",
      cash_out: "0.00",
      expected_balance: entries.length ? "120.00" : "100.00",
      counted_balance: shift?.closed_at ? "115.00" : null,
      difference: shift?.closed_at ? "-5.00" : null,
    };
  }
  page.on("request", (r) => {
    const u = new URL(r.url()),
      p = u.pathname;
    if (!p.startsWith("/api/")) return r.continue();
    const input = r.postData() ? JSON.parse(r.postData()) : null;
    calls.push({
      path: p,
      query: u.search,
      method: r.method(),
      headers: r.headers(),
      input,
    });
    if (p === "/api/auth/refresh")
      return actor === "anonymous"
        ? reply(r, {}, 401)
        : reply(r, {
            token: "fixture",
            user: {
              id: "user-1",
              role: actor,
              brand_id: "brand-1",
              tenant_id: actor === "cashier" ? tenant : undefined,
              email: "fixture@example.invalid",
            },
          });
    if (p.endsWith("/buyer-session"))
      return reply(r, {
        creation_key: "60000000-0000-4000-8000-000000000001",
        payment_key: "70000000-0000-4000-8000-000000000001",
      });
    if (p.startsWith("/api/public/menu/")) {
      const data = {
        location: {
          id: tenant,
          name: "Local Centro",
          brand_name: "Burger de barrio",
          address: "Av. de la cocina 125",
          phone: "11 5555 0123",
          hours: "Martes a domingo, 19 a 23",
        },
        currency: "ARS",
        categories: ["Hamburguesas", "Acompañamientos"],
        total: 2,
        items: menuItems,
      };
      if (u.searchParams.get("search")) {
        data.items = [menuItems[0]];
        data.total = 1;
      }
      if (u.searchParams.get("category")) {
        data.items = menuItems.filter(
          (x) => x.category === u.searchParams.get("category"),
        );
        data.total = data.items.length;
      }
      if (holdMenu) {
        pendingMenu = () => reply(r, data);
        return;
      }
      return reply(r, data);
    }
    if (p === "/api/tenants") return reply(r, [location]);
    if (p === "/api/business/locations") return reply(r, { items: [location] });
    if (p === "/api/business")
      return reply(r, { items: [{ id: "brand-1", name: "Burger de barrio" }] });
    if (p === "/api/products")
      return reply(r, [
        {
          ...menuItems[0],
          ID: product,
          base_price: "9800.00",
          is_active: true,
        },
      ]);
    if (p === "/api/tenant-products")
      return reply(r, [
        {
          product_id: product,
          tenant_id: tenant,
          product: { ...menuItems[0], ID: product, base_price: "9800.00" },
          is_active: true,
          is_available: true,
          track_stock: true,
          current_stock: 20,
        },
      ]);
    if (p === "/api/menu/settings") {
      if (r.method() === "PATCH") published = input.published;
      return reply(r, { published, path: "/catalogo/" + tenant });
    }
    if (p === "/api/inventory")
      return reply(r, {
        items: u.searchParams.get("search")
          ? [
              {
                id: "supply-1",
                kind: "supply",
                name: "Resultado remoto",
                current_stock: "1",
                minimum_stock: "2",
                unit: "kg",
                track_stock: true,
                stock_version: 1,
              },
            ]
          : [],
        total: 1,
      });
    if (p === "/api/cash/shifts" && r.method() === "GET")
      return reply(r, { items: shift ? [shift] : [] });
    if (p === "/api/cash/shifts" && r.method() === "POST") {
      shift = {
        id: "shift-1",
        currency: "ARS",
        opening_balance: "100.00",
        opened_at: "2026-11-12T10:00:00Z",
        closed_at: null,
      };
      return reply(r, shift, 201);
    }
    if (p === "/api/cash/shifts/shift-1") return reply(r, snapshot());
    if (p === "/api/cash/shifts/shift-1/entries") {
      if (r.method() === "POST") {
        entries.push({
          id: "entry-1",
          ...input,
          created_at: "2026-11-12T11:00:00Z",
        });
        return reply(r, entries.at(-1), 201);
      }
      return reply(r, { items: entries });
    }
    if (p === "/api/cash/shifts/shift-1/close") {
      shift = {
        ...shift,
        closed_at: "2026-11-12T12:00:00Z",
        counted_balance: "115.00",
        expected_balance: "120.00",
        difference: "-5.00",
        closing_note: input.note,
      };
      return reply(r, snapshot());
    }
    return reply(r, []);
  });
  return {
    page,
    errors,
    calls,
    setHold: () => (holdMenu = true),
    release: async () => {
      await page.waitForFunction(
        () => !!document.querySelector(".menu-product-skeleton"),
      );
      assert(pendingMenu);
      holdMenu = false;
      await pendingMenu();
    },
  };
}
async function clickText(page, text) {
  await page.waitForFunction(
    (text) =>
      [...document.querySelectorAll("button")].some(
        (b) => b.textContent.trim() === text && !b.disabled,
      ),
    {},
    text,
  );
  await page.evaluate(
    (text) =>
      [...document.querySelectorAll("button")]
        .find((b) => b.textContent.trim() === text && !b.disabled)
        .click(),
    text,
  );
}
async function fill(page, label, value) {
  await page.evaluate(
    ({ label, value }) => {
      const root = [...document.querySelectorAll("label")].find((e) =>
        e.textContent.includes(label),
      );
      const input = root?.querySelector("input,select,textarea");
      if (!input) throw new Error("Missing field " + label);
      input.value = value;
      input.dispatchEvent(
        new Event(input.tagName === "SELECT" ? "change" : "input", {
          bubbles: true,
        }),
      );
    },
    { label, value },
  );
}
async function loaded(page) {
  await page.waitForSelector("[data-dashboard-loading]", { hidden: true });
}
async function assertScreen({ page, errors }, label) {
  assert.deepEqual(errors, [], label);
  assert.equal(
    await page.evaluate(
      () => document.documentElement.scrollWidth > window.innerWidth + 1,
    ),
    false,
    label + " overflow",
  );
  assert.equal(await page.$(".app-progress"), null);
  assert.equal(await page.$(".motion-toggle"), null);
  await page.screenshot({
    path: ".cache/menu-cash-browser/" + label + ".png",
    fullPage: true,
  });
}
try {
  for (const width of [1440, 390]) {
    const theme = width === 1440 ? "dark" : "light";
    let state = await setup(width, theme, "anonymous");
    state.setHold();
    await state.page.goto(origin + "/catalogo/" + tenant, {
      waitUntil: "domcontentloaded",
    });
    await state.page.waitForSelector(".menu-product-skeleton");
    await state.page.evaluate(() => document.fonts.ready);
    const before = await state.page.$eval(
      ".menu-controls",
      (e) => e.getBoundingClientRect().y,
    );
    await state.release();
    await state.page.waitForSelector(
      ".menu-product:not(.menu-product-skeleton)",
    );
    const after = await state.page.$eval(
      ".menu-controls",
      (e) => e.getBoundingClientRect().y,
    );
    assert(
      Math.abs(after - before) <= 1,
      "Public menu header shifted while loading",
    );
    assert(
      (await state.page.$eval("main", (e) => e.textContent)).includes(
        "No disponible por ahora",
      ),
    );
    await fill(state.page, "Buscar en la carta", "doble");
    await state.page.waitForFunction(
      () =>
        document.querySelectorAll(".menu-product:not(.menu-product-skeleton)")
          .length === 1,
    );
    assert(
      state.calls.some(
        (c) =>
          c.path.includes("/public/menu/") && c.query.includes("search=doble"),
      ),
    );
    assert(
      state.calls
        .filter((c) => c.path.includes("/public/menu/"))
        .every((c) => !c.headers.authorization),
    );
    await assertScreen(state, "public-" + width);
    await state.page.close();
    state = await setup(width, theme);
    await state.page.goto(origin + "/admin/catalog/tenants", {
      waitUntil: "networkidle0",
    });
    await state.page.waitForSelector("#catalog-location");
    await state.page.select("#catalog-location", tenant);
    await state.page.waitForSelector(".menu-publication");
    await clickText(state.page, "Publicar carta");
    await state.page.waitForSelector(".menu-qr img");
    assert(
      (await state.page.$eval(".menu-qr img", (e) => e.src)).startsWith(
        "data:image/png;base64,",
      ),
    );
    assert.equal(
      await state.page.$eval(
        '.menu-publication a[target="_blank"]',
        (e) => e.href,
      ),
      origin + "/catalogo/" + tenant,
    );
    const publication = state.calls.find(
      (c) => c.path === "/api/menu/settings" && c.method === "PATCH",
    );
    assert.equal(publication.headers["x-tenant-id"], tenant);
    assert(publication.headers["idempotency-key"]);
    await assertScreen(state, "publication-" + width);
    await clickText(state.page, "Dejar de publicar");
    await state.page.waitForSelector(".menu-qr img", { hidden: true });
    await state.page.close();
    state = await setup(width, theme, "cashier");
    await state.page.goto(origin + "/cash", { waitUntil: "networkidle0" });
    await loaded(state.page);
    await fill(state.page, "Efectivo inicial", "100.00");
    await clickText(state.page, "Abrir caja en ARS");
    await state.page.waitForFunction(() =>
      document.body.textContent.includes("Ingreso o retiro"),
    );
    await loaded(state.page);
    await fill(state.page, "Movimiento", "cash_in");
    await fill(state.page, "Importe", "20.00");
    await fill(state.page, "Motivo", "Cambio para el turno");
    await clickText(state.page, "Registrar movimiento");
    await state.page.waitForFunction(() =>
      document.body.textContent.includes("Movimiento de efectivo registrado"),
    );
    await loaded(state.page);
    await clickText(state.page, "Arqueo y cierre");
    await fill(state.page, "Efectivo contado", "115.00");
    await fill(state.page, "Nota de cierre", "Faltante detectado al contar");
    await clickText(state.page, "Revisar y cerrar caja");
    await state.page.waitForSelector(".confirmation-card");
    await clickText(state.page, "Confirmar");
    await state.page.waitForSelector(".confirmation-card", { hidden: true });
    await state.page.waitForFunction(() =>
      document.body.textContent.includes("Caja cerrada"),
    );
    await loaded(state.page);
    assert.equal(
      state.calls.filter((c) => c.path.endsWith("/close")).length,
      1,
    );
    assert.equal(
      state.calls.find((c) => c.path.endsWith("/close")).input.counted_balance,
      "115.00",
    );
    assert(
      state.calls
        .filter((c) => c.method === "POST" && c.path.startsWith("/api/cash"))
        .every(
          (c) =>
            c.headers["x-tenant-id"] === tenant && c.headers["idempotency-key"],
        ),
    );
    await assertScreen(state, "cash-" + width);
    await state.page.close();
    state = await setup(width, theme);
    await state.page.goto(origin + "/inventory", { waitUntil: "networkidle0" });
    await loaded(state.page);
    await fill(state.page, "Buscar en todo el inventario", "needle");
    await state.page.waitForFunction(() =>
      document.body.textContent.includes("Resultado remoto"),
    );
    assert(
      state.calls.some(
        (c) => c.path === "/api/inventory" && c.query.includes("search=needle"),
      ),
    );
    await assertScreen(state, "inventory-" + width);
    await state.page.close();
  }
  console.log(
    "Menu/cash browser passed: public delayed menu without header shift, remote search, publication and local QR, withdrawal of publication, cashier opening/movement/reconciled closing and scoped idempotent requests at 1440/390.",
  );
} catch (e) {
  if (currentPage && !currentPage.isClosed()) {
    console.error(
      "Menu/cash browser failed",
      currentPage.url(),
      await currentPage.$eval("body", (e) => e.innerText),
    );
    await currentPage.screenshot({
      path: ".cache/menu-cash-browser/failure.png",
      fullPage: true,
    });
  }
  throw e;
} finally {
  await browser.close();
  await new Promise((r) => server.close(r));
}
