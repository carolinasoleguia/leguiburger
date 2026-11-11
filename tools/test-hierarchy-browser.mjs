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
await mkdir(".cache/hierarchy-browser", { recursive: true });
let role = "owner",
  selectedTenant = "tenant-1",
  suspended = false,
  failCreate = true,
  page;
const errors = [],
  captured = [];
const business = () => ({
  id: "brand-1",
  name: "Burger Fixture",
  owner_email: "owner@example.invalid",
  primary_owner_id: "fixture-user",
  suspended_at: suspended ? new Date().toISOString() : null,
  suspension_reason: suspended ? "Fixture suspension" : "",
  base_location_limit: 2,
  base_collaborator_limit: 5,
  maximum_locations: 2,
  maximum_collaborators: 5,
  locations: 1,
  collaborators: 1,
  invitations: [],
});
const locals = [
  {
    id: "tenant-1",
    brand_id: "brand-1",
    subdomain: "centro",
    display_name: "Centro",
    active: true,
  },
  {
    id: "tenant-2",
    brand_id: "brand-1",
    subdomain: "norte",
    display_name: "Norte",
    active: true,
  },
];
const member = {
  id: "employee-1",
  brand_id: "brand-1",
  tenant_id: "tenant-1",
  first_name: "Ana",
  last_name: "Equipo",
  email: "ana@example.invalid",
  role: "cashier",
  is_active: true,
  location_ids: ["tenant-1"],
};
const session = () => ({
  token: "fixture-access-token",
  user: {
    id: "fixture-user",
    role,
    brand_id: "brand-1",
    email: "fixture@example.invalid",
    tenant_id: role === "cashier" ? selectedTenant : "",
  },
});
try {
  page = await browser.newPage();
  page.on("dialog", dialog => dialog.accept());
  page.on("pageerror", (e) => errors.push(e.message));
  await page.setRequestInterception(true);
  page.on("request", (request) => {
    const path = new URL(request.url()).pathname;
    if (!path.startsWith("/api/")) return request.continue();
    const method = request.method(),
      body = JSON.parse(request.postData() || "{}");
    let data = {},
      status = 200;
    if (path === "/api/auth/refresh") data = session();
    else if (path === "/api/tenants") data = locals;
    else if (path === "/api/platform/restaurants" && method === "POST") {
      captured.push({
        type: "create",
        body,
        key: request.headers()["idempotency-key"],
      });
      if (failCreate) {
        status = 503;
        data = { message: "Respuesta incierta: reintentá." };
        failCreate = false;
      } else
        data = {
          brand_id: "brand-1",
          tenant_id: "tenant-1",
          invitation: {
            invitation_url:
              origin + "/accept-business-invitation#token=owner-capability",
          },
        };
    } else if (path === "/api/platform/restaurants" || path === "/api/business")
      data = { items: [business()] };
    else if (path === "/api/platform/restaurants/brand-1/status") {
      captured.push({ type: "status", body });
      suspended = body.suspended;
      data = { ok: true };
    } else if (path === "/api/business/brand-1/owner-invitation") {
      captured.push({ type: "transfer", body });
      data = {
        invitation_url:
          origin + "/accept-business-invitation#token=transfer-capability",
      };
    } else if (path === "/api/business/locations" && method === "POST") {
      captured.push({ type: "location", body });
      data = { id: "tenant-3" };
    } else if (path === "/api/business/locations") data = { items: locals };
    else if (path === "/api/business/team") data = { items: [member] };
    else if (path === "/api/business/team/employee-1") {
      captured.push({ type: "team", body });
      Object.assign(member, body);
      data = { ok: true };
    } else if (path === "/api/auth/location") {
      captured.push({ type: "context", body });
      selectedTenant = body.tenant_id;
      data = session();
    } else if (path === "/api/auth/business-invitations/accept") {
      captured.push({ type: "accept", body });
      data = { message: "Tu restaurante está listo. Ya podés iniciar sesión." };
    } else if (path === "/api/payments/mercadopago/config")
      data = { enabled: false, environment: "sandbox" };
    else if (path === "/api/billing/my-account")
      data = { subscriptions: [], invoices: [], accounts: [] };
    else if (
      path === "/api/billing/plans" ||
      path === "/api/orders" ||
      path === "/api/pos/catalog" ||
      path === "/api/crm/customers"
    )
      data = { items: [] };
    else {
      status = 404;
      data = { message: "Unexpected mock route " + path };
    }
    request.respond({
      status,
      contentType: "application/json",
      body: JSON.stringify(data),
    });
  });
  const goto = async (path) => {
    await page.goto(origin + path, { waitUntil: "networkidle0" });
    await page.waitForSelector("main");
  };
  const click = async (text) =>
    page.evaluate((text) => {
      const item = [...document.querySelectorAll("button")].find(
        (x) => x.textContent.trim() === text,
      );
      if (!item) throw new Error("Missing button " + text);
      item.click();
    }, text);
  const fill = async (label, value) =>
    page.evaluate(
      ({ label, value }) => {
        const labels = [...document.querySelectorAll("label")];
        const element = (
          labels.find((x) => x.firstChild?.textContent.trim() === label) ||
          labels.find((x) => x.textContent.trim().startsWith(label))
        )?.querySelector("input,select");
        if (!element) throw new Error("Missing field " + label);
        element.value = value;
        element.dispatchEvent(
          new Event(element.tagName === "SELECT" ? "change" : "input", {
            bubbles: true,
          }),
        );
      },
      { label, value },
    );
  await goto("/restaurants");
  await fill("Nombre del negocio", "Burger nuevo");
  await fill("Correo del dueño", "new-owner@example.invalid");
  await click("Crear restaurante e invitar dueño");
  await page.waitForSelector('[role="alert"]');
  await click("Crear restaurante e invitar dueño");
  await page.waitForFunction(() =>
    document
      .querySelector("input[readonly]")
      ?.value.includes("owner-capability"),
  );
  assert.equal(captured[0].key, captured[1].key);
  assert(
    !("password" in captured[1].body),
    "platform must not pick the owner password",
  );
  await goto("/locations");
  await fill("Identificador", "sur");
  await click("Guardar local");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Local guardado"),
  );
  assert.equal(
    captured.find((x) => x.type === "location").body.brand_id,
    "brand-1",
  );
  await goto("/team");
  await click("Administrar acceso");
  assert.equal(
    await page.$$eval(
      "select option",
      (all) => all.filter((x) => x.value === "admin").length,
    ),
    0,
  );
  await page.evaluate(() => {
    const input = document.querySelector(
      'input[type="checkbox"][value="tenant-2"]',
    );
    input.click();
  });
  await click("Guardar accesos");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("Colaborador actualizado"),
  );
  assert.deepEqual(captured.find((x) => x.type === "team").body.location_ids, [
    "tenant-1",
    "tenant-2",
  ]);
  for (const width of [1440, 390]) {
    await page.setViewport({ width, height: 960 });
    for (const route of ["/restaurants", "/locations", "/team"]) {
      await goto(route);
      assert(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
        `${route} overflow at ${width}`,
      );
      await page.screenshot({
        path: `.cache/hierarchy-browser/${route.slice(1)}-${width}.png`,
        fullPage: true,
      });
    }
  }
  role = "admin";
  await goto("/restaurants");
  assert.equal(
    await page.$$eval(
      "button",
      (all) =>
        all.filter((x) => x.textContent.includes("Crear restaurante")).length,
    ),
    0,
  );
  await click("Transferir responsabilidad");
  await fill("Correo del responsable", "next@example.invalid");
  await fill("Tu contraseña actual", "fixture-current-password");
  await click("Generar invitación");
  await page.waitForFunction(() =>
    document
      .querySelector("input[readonly]")
      ?.value.includes("transfer-capability"),
  );
  assert.equal(
    captured.find((x) => x.type === "transfer").body.current_password,
    "fixture-current-password",
  );
  suspended = true;
  await goto("/payments");
  assert(
    await page.$eval("main", (x) =>
      x.textContent.includes("restaurante está suspendido"),
    ),
  );
  assert.equal(await page.$$eval(".payment-tabs button", (x) => x.length), 1);
  suspended = false;
  role = "cashier";
  await goto("/orders");
  await fill("Local de trabajo", "tenant-2");
  await page.waitForFunction(
    () => document.querySelector("select")?.value === "tenant-2",
  );
  assert.equal(
    captured.find((x) => x.type === "context").body.tenant_id,
    "tenant-2",
  );
  await goto("/accept-business-invitation#token=owner-capability");
  assert.equal(new URL(page.url()).hash, "");
  await page.type("#first-name", "Nuevo");
  await page.type("#last-name", "Dueño");
  await page.type("#new-password", "fixture-new-password");
  await page.type("#confirm-password", "fixture-new-password");
  await click("Guardar y continuar");
  await page.waitForFunction(() =>
    document
      .querySelector('[role="status"]')
      ?.textContent.includes("restaurante está listo"),
  );
  assert.deepEqual(errors, []);
  console.log(
    "Hierarchy browser passed: password-free onboarding, idempotent retries, local creation, role-safe multi-location team, ownership transfer, suspended membership access, context switch, invitation acceptance and 1440/390 layouts.",
  );
} catch (e) {
  console.error("Browser runtime errors", errors, "URL", page?.url());
  console.error(await page?.evaluate(() => document.body.innerText));
  await page?.screenshot({
    path: ".cache/hierarchy-browser/failure.png",
    fullPage: true,
  });
  throw e;
} finally {
  await browser.close();
  await new Promise((resolve) => server.close(resolve));
}
