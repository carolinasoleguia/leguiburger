import assert from 'node:assert/strict';
import { mkdir, readFile } from 'node:fs/promises';
import { createServer } from 'node:http';
import { resolve, sep, extname } from 'node:path';
import { pathToFileURL } from 'node:url';

const modulePath = process.env.PUPPETEER_MODULE;
if (!modulePath) throw new Error('Set PUPPETEER_MODULE to the installed Puppeteer module');
const { default: puppeteer } = await import(pathToFileURL(modulePath).href);
const dist = resolve('frontend/dist');
const server = createServer(async (request, response) => {
  let file = resolve(dist, '.' + new URL(request.url, 'http://localhost').pathname);
  if (!file.startsWith(dist + sep) || !extname(file)) file = resolve(dist, 'index.html');
  try { const body = await readFile(file); response.setHeader('Content-Type', ({ '.html': 'text/html', '.js': 'application/javascript', '.css': 'text/css', '.woff2': 'font/woff2', '.svg': 'image/svg+xml' })[extname(file)] || 'application/octet-stream'); response.end(body); }
  catch { response.writeHead(404); response.end(); }
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const origin = process.env.SECURITY_PREVIEW_URL || `http://127.0.0.1:${server.address().port}`;
await mkdir('.cache/security-browser', { recursive: true });
const browser = await puppeteer.launch({ headless: true, executablePath: process.env.CHROME_PATH, args: ['--disable-gpu', ...(process.env.CI ? ['--no-sandbox'] : [])] });
const user = role => ({ id: 'fixture-user', role, email: `${role}@example.invalid`, brand_id: role === 'owner' ? '' : 'fixture-brand', tenant_id: role === 'employee' ? 'fixture-tenant' : '' });
let active = null;
let freshLogin = null;
let tokenIndex = 0;
const sessionPayload = role => ({ token: `fixture-memory-token-${++tokenIndex}`, user: user(role), session_id: 'fixture-session', expires_in: 900 });
const errors = [];
try {
  const page = await browser.newPage();
  page.on('pageerror', error => errors.push(error.message));
  await page.setRequestInterception(true);
  page.on('request', request => {
    const path = new URL(request.url()).pathname;
    if (!path.startsWith('/api/')) return request.continue();
    const body = JSON.parse(request.postData() || '{}');
    let status = 200, result = {}, headers = {};
    const grant = role => { active = role; headers['set-cookie'] = 'legui_refresh=fixture-cookie; HttpOnly; SameSite=Strict; Path=/api/auth'; return sessionPayload(role); };
    if (path === '/api/auth/refresh') { if (active) result = grant(active); else { status = 401; result = { message: 'Sesión vencida' }; } }
    else if (path === '/api/auth/login') { freshLogin = body.email.split('@')[0]; result = freshLogin === 'owner' ? { mfa_required: true, enrollment_required: true, challenge: 'fixture-challenge' } : grant(freshLogin); }
    else if (path === '/api/auth/mfa/enroll') result = { secret: 'GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ' };
    else if (path === '/api/auth/mfa/verify') result = { ...grant('owner'), recovery_codes: ['recovery-fixture-one', 'recovery-fixture-two'] };
    else if (path === '/api/auth/logout' || path === '/api/auth/logout-all') { active = null; headers['set-cookie'] = 'legui_refresh=; Max-Age=0; HttpOnly; SameSite=Strict; Path=/api/auth'; result = { revoked: true }; }
    else if (path === '/api/auth/password/change') { active = null; result = { message: 'Contraseña actualizada' }; }
    else if (path === '/api/auth/password/forgot') result = { message: 'Si existe una cuenta con ese correo, recibirás un enlace para restablecer tu contraseña.' };
    else if (path === '/api/auth/password/reset') result = { message: 'Contraseña actualizada. Iniciá sesión nuevamente.' };
    else if (path === '/api/auth/invitations/accept') { status = 201; result = { message: 'Tu cuenta está lista. Ya podés iniciar sesión.' }; }
    else if (path === '/api/auth/sessions') result = [{ id: 'fixture-session', current: true, user_agent: 'Chrome fixture', last_seen_at: new Date().toISOString() }];
    else if (path === '/api/team/invitations' && request.method() === 'POST') { status = 201; result = { id: 'fixture-invite', invitation_url: `${origin}/accept-invitation#token=fixture-invite` }; }
    else if (path === '/api/team/invitations') result = [];
    else if (path === '/api/audit') result = { items: [{ id: 'fixture-event', occurred_at: new Date().toISOString(), actor_name: 'Persona de prueba', event_name: 'team.invited', target_id: 'fixture-invite' }] };
    else if (path === '/api/tenants') result = [{ id: 'fixture-tenant', subdomain: 'Centro', brand_id: 'fixture-brand', active: true, brand: { id: 'fixture-brand', name: 'Hamburguesería' } }];
    else result = [];
    request.respond({ status, headers, contentType: 'application/json', body: JSON.stringify(result) });
  });
  const goto = async path => {await page.goto(origin + path, { waitUntil: 'networkidle0' });await page.waitForFunction(()=>document.querySelector('#app')?.childElementCount>0);};
  const submit = async (email) => { await page.type('#email', email); await page.type('#password', 'fixture-password-123'); await page.click('form button.btn-submit'); };
  await goto('/login'); await submit('owner@example.invalid');
  await page.waitForSelector('#mfa-code'); assert(await page.$('.mfa-setup'));
  await page.type('#mfa-code', '123456'); await page.click('form button.btn-submit');
  await page.waitForSelector('.recovery-codes');
  assert(await page.$eval('.recovery-codes', element => element.textContent.includes('recovery-fixture-one')));
  await page.click('.login-card > .btn-submit');
  await page.waitForFunction(() => location.pathname === '/owner');
  assert.equal(await page.evaluate(() => localStorage.getItem('token')), null);
  assert.equal(await page.evaluate(() => document.cookie.includes('legui_refresh')), false);
  await page.reload({ waitUntil: 'networkidle0' }); assert.equal(new URL(page.url()).pathname, '/owner');
  await goto('/account/security'); await page.waitForSelector('#current-password');
  for (const width of [1440, 390]) {
    await page.setViewport({ width, height: 960 });
    assert(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth), `Security page overflows at ${width}`);
    await page.screenshot({ path: `.cache/security-browser/account-${width}.png`, fullPage: true });
  }
  await page.setViewport({ width: 1440, height: 960 });
  await page.type('#current-password', 'fixture-password-123'); await page.type('#security-password', 'new-fixture-password-123'); await page.type('#security-confirm', 'new-fixture-password-123');
  await page.click('form button.btn-primary'); await page.waitForSelector('#email');
  await submit('admin@example.invalid'); await page.waitForFunction(() => location.pathname === '/admin');
  await goto('/team/invitations'); await page.type('#invite-email', 'collaborator@example.invalid'); await page.click('form button.btn-primary'); await page.waitForSelector('#invitation-link');
  assert(await page.$eval('#invitation-link', input => input.value.includes('/accept-invitation#token=')));
  await goto('/audit'); assert(await page.$eval('tbody', table => table.textContent.includes('Invitación enviada')));
  await page.click('.btn-logout'); await page.waitForSelector('#email');
  await submit('employee@example.invalid'); await page.waitForFunction(() => location.pathname.includes('/employee'));
  await goto('/audit'); assert.equal(new URL(page.url()).pathname, '/unauthorized');
  await goto('/account/security');
  await page.evaluate(() => localStorage.setItem('legui-last-activity', String(Date.now() - 31 * 60 * 1000)));
  await page.waitForFunction(() => location.pathname === '/login', { timeout: 15000 });
  await goto('/forgot-password'); await page.type('#recovery-email', 'employee@example.invalid'); await page.click('form button');
  await page.waitForFunction(() => document.querySelector('[role="status"]')?.textContent.includes('Si existe'));
  await goto('/reset-password#token=fixture-reset-token'); assert.equal(new URL(page.url()).hash, '');
  await page.type('#new-password', 'fixture-new-password-123'); await page.type('#confirm-password', 'fixture-new-password-123'); await page.click('form button');
  await page.waitForFunction(() => document.querySelector('[role="status"]')?.textContent.includes('actualizada'));
  await goto('/accept-invitation#token=fixture-invitation'); await page.type('#first-name', 'Prueba'); await page.type('#last-name', 'Equipo');
  await page.type('#new-password', 'fixture-new-password-123'); await page.type('#confirm-password', 'fixture-new-password-123'); await page.click('form button');
  await page.waitForFunction(() => document.querySelector('[role="status"]')?.textContent.includes('cuenta está lista'));
  assert.deepEqual(errors, []);
  console.log('Security browser checks passed: MFA enrollment/recovery UI, memory-only token, HttpOnly cookie, reload/refresh, sessions/password change, invitations, audit, collaborator routing, idle logout and recovery/acceptance screens (1440px and 390px).');
} finally { await browser.close(); await new Promise(resolve => server.close(resolve)); }
