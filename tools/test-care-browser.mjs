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
await mkdir(".cache/care-browser", { recursive: true });
let page,
  role = "admin",
  failedOnce = true,
  archived = false,
  backupCreated = false;
const errors = [],
  requests = [],
  tickets = [],
  messages = [];
try {
  page = await browser.newPage();
  page.on("pageerror", (error) => errors.push(error.message));
  page.on("dialog", (dialog) => dialog.accept());
  await page.setViewport({ width: 1440, height: 1050 });
  await page.setRequestInterception(true);
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (!path.startsWith("/api/")) return request.continue();
    const method = request.method(),
      body = JSON.parse(request.postData() || "{}");
    let data = [],
      status = 200;
    requests.push({
      path,
      method,
      body,
      key: request.headers()["idempotency-key"],
    });
    if (path === "/api/auth/refresh")
      data = {
        token: "fixture-token",
        user: {
          id: "user-1",
          role,
          brand_id: "brand-1",
          email: "owner@example.invalid",
        },
      };
    else if (path === "/api/tenants")
      data = [
        {
          id: "tenant-1",
          brand_id: "brand-1",
          name: "Centro",
          subdomain: "centro",
          active: true,
        },
      ];
    else if (path === "/api/business" || path === "/api/platform/restaurants")
      data = { items: [{ id: "brand-1", name: "Burger de barrio" }] };
    else if (path === "/api/care/support" && method === "POST") {
      if (failedOnce) {
        failedOnce = false;
        status = 503;
        data = {
          message: "Respuesta incierta. Reintentá sin perder el mensaje.",
        };
      } else {
        tickets.push({
          id: "ticket-1",
          subject: body.subject,
          state: "open",
          updated_at: new Date().toISOString(),
        });
        messages.push({
          id: "msg-1",
          body: body.body,
          platform: false,
          created_at: new Date().toISOString(),
        });
        data = { id: "ticket-1" };
      }
    } else if (path === "/api/care/support") data = tickets;
    else if (path === "/api/care/support/ticket-1") data = messages;
    else if (path === "/api/care/support/ticket-1/messages") {
      messages.push({
        id: "msg-2",
        body: body.body,
        platform: role === "owner",
        created_at: new Date().toISOString(),
      });
      data = { saved: true };
    } else if (path === "/api/care/backups" && method === "POST") {
      backupCreated = true;
      data = { id: "backup-1", state: "pending" };
    } else if (path === "/api/care/backups")
      data = backupCreated
        ? [
            {
              id: "backup-1",
              state: "ready",
              report: { verified: true },
              created_at: new Date().toISOString(),
              verified_at: new Date().toISOString(),
            },
          ]
        : [];
    else if (path === "/api/care/closure" && method === "POST") {
      archived = true;
      data = { archived: true, provider_cancellations_pending: 1 };
    } else if (path === "/api/care/closure")
      data = {
        name: "Burger de barrio",
        primary: true,
        closed_at: archived ? new Date().toISOString() : null,
        renewals: archived
          ? [
              {
                id: "sub-1",
                provider: "mercadopago",
                cancel_pending: true,
                operation_state: "review",
              },
            ]
          : [],
      };
    else if (path === "/api/care/errors")
      data =
        method === "POST"
          ? { received: true }
          : [
              {
                component: "web",
                code: "render",
                occurrences: 2,
                last_seen_at: new Date().toISOString(),
              },
            ];
    request.respond({
      status,
      contentType: "application/json",
      body: JSON.stringify(data),
    });
  });
  const goto = async (path) => {
    await page.goto(origin + path, { waitUntil: "networkidle0" });
    await page.waitForFunction(
      () => document.querySelector("#app")?.childElementCount > 0,
    );
    await page.waitForNetworkIdle({ idleTime: 100 });
  };
  const click = async (text) => {
    await page.waitForFunction(
      (text) =>
        [...document.querySelectorAll("button")].some(
          (b) => b.textContent.includes(text) && !b.disabled,
        ),
      {},
      text,
    );
    await page.evaluate(
      (text) =>
        [...document.querySelectorAll("button")]
          .find((b) => b.textContent.includes(text) && !b.disabled)
          .click(),
      text,
    );
  };
  const fill = async (text, value) => {
    await page.evaluate(
      (text, value) => {
        const label = [...document.querySelectorAll("label")].find((l) =>
            l.textContent.includes(text),
          ),
          input = label?.querySelector("input,textarea,select");
        if (!input) throw new Error("missing field " + text);
        input.value = value;
        input.dispatchEvent(new Event("input", { bubbles: true }));
        input.dispatchEvent(new Event("change", { bubbles: true }));
      },
      text,
      value,
    );
  };
  await goto("/care");
  await fill("Asunto", "Consulta de mi restaurante");
  await fill(
    "¿En qué podemos ayudarte?",
    "No se confirmó la operación de prueba",
  );
  await click("Enviar consulta");
  await page.waitForSelector('[role="alert"]');
  assert.equal(
    await page.$eval('input[name="subject"]', (el) => el.value),
    "Consulta de mi restaurante",
  );
  await click("Enviar consulta");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Consulta enviada"),
  );
  const attempts = requests.filter(
    (r) => r.path === "/api/care/support" && r.method === "POST",
  );
  assert.equal(attempts.length, 2);
  assert.equal(attempts[0].key, attempts[1].key);
  await fill("Tu respuesta", "Agrego información útil para soporte");
  await click("Enviar respuesta");
  await page.waitForFunction(() =>
    document.body.textContent.includes("Agrego información útil para soporte"),
  );
  await click("Crear respaldo verificado");
  await page.waitForFunction(() =>
    document.body.textContent.includes("Reconstrucción validada"),
  );
  await fill("Asunto", "Borrador sin guardar");
  await page.evaluate(() =>
    [...document.querySelectorAll('a[href="/restaurants"]')]
      .find((el) => el.getClientRects().length)
      .click(),
  );
  await page.waitForSelector(".confirmation-card");
  assert.equal(new URL(page.url()).pathname, "/care");
  await click("Cancelar");
  assert.equal(
    await page.$eval('input[name="subject"]', (el) => el.value),
    "Borrador sin guardar",
  );
  // Keep the draft while cancelling navigation; native beforeunload confirms a
  // deliberate full-page test navigation, as it does for the other browser suites.
  await goto("/care");
  await fill("Motivo", "Cierre definitivo de prueba");
  await fill("Contraseña actual", "fixture-password-only");
  await fill("Escribí CERRAR RESTAURANTE", "CERRAR RESTAURANTE");
  await click("Archivar restaurante");
  await page.waitForFunction(() =>
    document.body.textContent.includes("Restaurante archivado"),
  );
  assert(
    await page.$eval("main", (el) =>
      el.textContent.includes("Pendiente de confirmación del proveedor"),
    ),
  );
  assert(
    await page.$eval("main", (el) =>
      el.textContent.includes("requiere conciliación"),
    ),
  );
  for (const width of [1440, 390]) {
    await page.setViewport({ width, height: 1050 });
    for (const theme of ["dark", "light"]) {
      await page.evaluate(
        (theme) => (document.documentElement.dataset.theme = theme),
        theme,
      );
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await page.screenshot({
        path: `.cache/care-browser/care-${width}-${theme}.png`,
        fullPage: true,
      });
    }
  }
  role = "owner";
  await goto("/care");
  assert(
    await page.$eval("main", (el) =>
      el.textContent.includes("Errores de la aplicación"),
    ),
  );
  role = "cashier";
  await goto("/care");
  assert.equal(await page.$$eval(".care-danger", (els) => els.length), 0);
  assert(
    !(await page.$eval("main", (el) =>
      el.textContent.includes("Crear respaldo"),
    )),
  );
  role = "admin";
  await page.setViewport({ width: 1440, height: 1050 });
  await goto("/admin/catalog");
  await click("Nuevo producto");
  await page.waitForSelector('[role="dialog"]');
  const dialog = await page.$('[role="dialog"]');
  assert.equal(
    await dialog.evaluate((el) => el.getAttribute("aria-modal")),
    "true",
  );
  assert(
    await page.evaluate(() =>
      document
        .querySelector('[role="dialog"]')
        .contains(document.activeElement),
    ),
  );
  await page.keyboard.press("Escape");
  await page.waitForFunction(() => !document.querySelector('[role="dialog"]'));
  for (const path of ["/admin/catalog", "/admin/production"]) {
    await goto(path);
    assert.equal(await page.$$eval("main h1", (els) => els.length), 1);
    for (const width of [1440, 390]) {
      await page.setViewport({ width, height: 1050 });
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth + 1,
        ),
      );
      await page.screenshot({
        path: `.cache/care-browser/${path.split("/").at(-1)}-${width}.png`,
        fullPage: true,
      });
    }
  }
  assert.deepEqual(errors, []);
  console.log(
    "Care browser passed: retained failed draft, stable retry key, support history, private backup state, unsaved navigation/cancel, closure reconciliation, role restrictions, modal keyboard/focus, legacy consistency and responsive themes.",
  );
} catch (error) {
  console.error("Care browser failure", page?.url(), errors);
  console.error(await page?.evaluate(() => document.body.innerText));
  await page?.screenshot({
    path: ".cache/care-browser/failure.png",
    fullPage: true,
  });
  throw error;
} finally {
  await browser.close();
  await new Promise((resolve) => server.close(resolve));
}
