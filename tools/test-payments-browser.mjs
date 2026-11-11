import assert from 'node:assert/strict';
import { createServer } from 'node:http';
import { readFile, mkdir } from 'node:fs/promises';
import { resolve, sep, extname } from 'node:path';
import { pathToFileURL } from 'node:url';
if (!process.env.PUPPETEER_MODULE) throw new Error('Set PUPPETEER_MODULE and CHROME_PATH');
const { default: puppeteer } = await import(pathToFileURL(process.env.PUPPETEER_MODULE).href);
const dist = resolve('frontend/dist');
const server = createServer(async (request, response) => {
  let file = resolve(dist, '.' + new URL(request.url, 'http://localhost').pathname);
  if (!file.startsWith(dist + sep) || !extname(file)) file = resolve(dist, 'index.html');
  try { const content = await readFile(file); response.setHeader('Content-Type', ({ '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.woff2': 'font/woff2', '.svg': 'image/svg+xml' })[extname(file)] || 'application/octet-stream'); response.end(content); } catch { response.writeHead(404); response.end(); }
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = `http://127.0.0.1:${server.address().port}`;
const browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH, args: ['--disable-gpu', ...(process.env.CI ? ['--no-sandbox'] : [])] });
await mkdir('.cache/payments-browser', { recursive: true });
let active = false, enabled = true, captured = [], refunded = false;
let debugPage;const requestLog=[],consoleLog=[];
const errors = [];
const session = () => ({ token: 'fixture-memory-token', user: { id: 'fixture-admin', role: 'admin', email: 'admin@example.invalid', brand_id: 'brand-1' } });
try {
  const page = await browser.newPage();
  page.on("dialog", dialog => dialog.accept()); await page.setViewport({width:1440,height:960}); page.on('pageerror', error => errors.push(error.message));
  debugPage=page;page.on('console',message=>{if(message.type()==='error'||message.type()==='warn')consoleLog.push(message.text())});
  await page.setRequestInterception(true);
  page.on('request', request => {
    const path = new URL(request.url()).pathname;
    if (!path.startsWith('/api/')) return request.continue();
    requestLog.push({path,method:request.method()});
    const body = JSON.parse(request.postData() || '{}'); let result = {}, status = 200;
    if (path === '/api/auth/refresh') { if (active) result = session(); else status = 401; }
    else if (path === '/api/auth/login') { active = true; result = session(); }
    else if (path === '/api/tenants') result = [{ id: 'tenant-1', brand_id: 'brand-1', subdomain: 'Centro', active: true }];
    else if (path === '/api/payments/mercadopago/config') result = { enabled, environment: 'sandbox', oauth_configured: enabled, platform_configured: enabled };
    else if (path === '/api/integrations/mercadopago/connections') result = { items: [{ id: 'connection-1', scope: 'business', environment: 'sandbox', state: 'connected', collector_id: '111' }] };
    else if (path === '/api/payments/mercadopago/checkouts') result = { items: [] };
    else if (path === '/api/payments/mercadopago/transactions') result = { items: [{ id: 'payment-1', payment_id: '1001', environment: 'sandbox', scope: 'business', provider_status: 'approved', allocated: true, amount: '200.00', refunded_amount: refunded ? '20.00' : '0.00', fee_amount: null, net_received: '190.00', currency: 'ARS', paid_at: new Date().toISOString() }] };
    else if (path === '/api/payments/mercadopago/reviews') result = { items: [{ id: 'review-1', resource_id: '1002', reason: 'AMOUNT_MISMATCH', last_seen_at: new Date().toISOString() }] };
    else if (path === '/api/billing/plans') result = { items: [{ id: 'price-1', name: 'Plan de prueba', amount: '1000', currency: 'ARS', billing_unit: 'brand', interval_months: 1 }] };
    else if (path === '/api/billing/my-account') result = { accounts: [], subscriptions: [], invoices: [] };
    else if (path === '/api/orders') result = { items: [{ id: 'order-fixture-1', total_due: '200', total_paid: '0', status: 'pending', currency: 'ARS' }] };
    else if (path === '/api/orders/order-fixture-1/checkout') { captured.push({ type: 'checkout', body }); result = { state: 'created', checkout_url: 'https://www.mercadopago.com.ar/checkout?fixture=order' }; }
    else if (path === '/api/billing/subscribe') { captured.push({ type: 'subscribe', body }); result = { subscription_id: 'subscription-1' }; }
    else if (path === '/api/billing/subscriptions/subscription-1/checkout') result = { state: 'created', checkout_url: 'https://www.mercadopago.com.ar/subscriptions/checkout?fixture=membership' };
    else if (path === '/api/payments/mercadopago/transactions/payment-1/refunds') { captured.push({ type: 'refund', body }); refunded = true; result = { state: 'completed' }; }
    else if (path === '/api/payments/mercadopago/reviews/review-1/resolve') { captured.push({ type: 'review', body }); result = { resolved: true }; }
    else result = [];
    request.respond({ status, contentType: 'application/json', body: JSON.stringify(result) });
  });
  const clickText = async text => {
    await page.waitForFunction(text=>{const button=[...document.querySelectorAll('button')].find(item=>item.textContent.trim()===text);return button&&!button.disabled;},{},text);
    await page.evaluate(text => { const button = [...document.querySelectorAll('button')].find(item => item.textContent.trim() === text); if (!button) throw new Error(`Missing button: ${text}`); button.click(); }, text);
  };
  await page.goto(origin + '/login', { waitUntil: 'networkidle0' });
  await page.waitForSelector('#email');
  await page.evaluate(()=>document.fonts.ready);
  await page.type('#email', 'admin@example.invalid'); await page.type('#password', 'fixture-password-123');
  assert(await page.$eval('form',form=>form.checkValidity()),'Fixture login must satisfy browser validation');
  await page.$eval('form',form=>form.requestSubmit());
  await page.waitForFunction(() => location.pathname === '/admin');
  await page.goto(origin + '/payments', { waitUntil: 'networkidle0' });
  assert(await page.$eval('.payment-table', table => table.textContent.includes('Por confirmar')));
  await clickText('Devolver'); await page.type('#refund-amount', '20.00'); await clickText('Confirmar devolución');
  await page.waitForFunction(() => document.querySelector('[role="status"]')?.textContent.includes('Devolución confirmada'));
  assert.equal(typeof captured.find(item => item.type === 'refund').body.amount, 'string');
  await clickText('Links de cobro'); await page.type('#partial-payment', '50.00'); await clickText('Crear link de cobro');
  await page.waitForSelector('#checkout-link'); assert.equal(typeof captured.find(item => item.type === 'checkout').body.amount, 'string');
  assert(await page.$eval('#checkout-link', input => input.value.includes('mercadopago.com.ar')));
  await clickText('Mi membresía'); await clickText('Continuar con Mercado Pago');
  await page.waitForFunction(() => document.querySelector('#checkout-link')?.value.includes('membership'));
  assert.equal(captured.find(item => item.type === 'subscribe').body.price_id, 'price-1');
  await clickText('Revisiones (1)'); await clickText('Revisar'); await page.type('#review-note', 'Importe contrastado con el proveedor.'); await clickText('Guardar resolución');
  await page.waitForFunction(() => document.querySelector('[role="status"]')?.textContent.includes('Resolución registrada'));
  assert.equal(captured.find(item => item.type === 'review').body.resolution, 'reprocess');
  for (const width of [1440, 390]) {
    await page.setViewport({ width, height: 960 });
    for (const tab of ['Movimientos', 'Links de cobro', 'Cuentas conectadas', 'Mi membresía', 'Revisiones (1)']) { await clickText(tab); assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `Payments overflow: ${tab} ${width}`); }
    await page.screenshot({ path: `.cache/payments-browser/payments-${width}.png`, fullPage: true });
  }
  enabled = false; await page.reload({ waitUntil: 'networkidle0' });
  assert(await page.$('.setup-notice'));
  await clickText('Links de cobro'); assert(await page.$eval('form button.btn-primary', button => button.disabled));
  await page.goto(origin + '/payments/return?status=approved', { waitUntil: 'networkidle0' });
  assert(await page.$eval('main', element => element.textContent.includes('no confirma por sí solo')));
  assert.deepEqual(errors, []);
  console.log('Payments browser checks passed: transactions/unknown costs, exact decimal inputs, partial checkout, refund, membership, reviews, disabled setup and honest return screen at 1440px/390px.');
} catch (error) { console.error('Browser runtime errors:', errors,consoleLog,'Requests:',requestLog,'Active:',active,'URL:',debugPage?.url());console.error(await debugPage?.evaluate(()=>document.body.innerText));throw error; }
finally { await browser.close(); await new Promise(resolve => server.close(resolve)); }
