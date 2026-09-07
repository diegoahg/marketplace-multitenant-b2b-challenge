import { test, expect } from '@playwright/test';

for (const [country, label, currency, total] of [
  ['CO', 'Colombia · COP', 'COP', 'COP 11,55'],
  ['EC', 'Ecuador · USD', 'USD', 'USD 11,55'],
  ['GT', 'Guatemala · GTQ', 'GTQ', 'Q 11,55'],
  ['AR', 'Argentina · ARS', 'ARS', 'ARS 11,55'],
]) {
  test(`quote and confirm in ${country}/${currency}`, async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await openShop(page);
    await page.getByText('Perú · PEN', { exact: true }).click();
    await page.getByRole('menuitem', { name: label, exact: true }).click();
    await page.getByRole('button', { name: 'Por volumen · 12 uds.', exact: true }).click();
    await page.getByRole('button', { name: /Ir a tu pedido/ }).click();
    await page.getByText('Contado', { exact: true }).click();
    const response = page.waitForResponse(r => r.url().endsWith('/api/quotes'));
    await page.getByRole('button', { name: 'Cotizar pedido', exact: true }).click();
    const q = await (await response).json();
    expect(q.country).toBe(country);
    expect(q.total).toEqual({ amount: '1155', currency, scale: 2 });
    expect(q.gifts[0].quantity).toBe(2);
    // Demo taxes are zero: taxable base and final total both show this amount.
    const displayedAmounts = page.getByText(total, { exact: true });
    await expect(displayedAmounts).toHaveCount(2);
    await expect(displayedAmounts.nth(0)).toBeVisible();
    await expect(displayedAmounts.nth(1)).toBeVisible();
    const confirmed = page.waitForResponse(r => r.url().endsWith('/api/orders') && r.request().method() === 'POST');
    await page.getByRole('button', { name: 'Confirmar pedido', exact: true }).click();
    const result = await confirmed;
    expect(result.status()).toBe(201);
    expect((await result.json()).total).toEqual(q.total);
  });
}

const browserErrors = new WeakMap();
test.beforeEach(async ({ page }) => {
  const errors = [];
  browserErrors.set(page, errors);
  page.on('pageerror', error => errors.push(error.message));
  page.on('console', message => {
    // The first test deliberately aborts the committed order's HTTP response.
    if (message.type() === 'error' && message.text() !== 'Failed to load resource: net::ERR_FAILED') {
      errors.push(message.text());
    }
  });
});
test.afterEach(async ({ page }) => {
  expect(browserErrors.get(page), 'No Flutter rendering errors in the browser console').toEqual([]);
});

async function openShop(page) {
  await page.goto('/');
  const semantics = page.locator('flt-semantics-placeholder');
  await semantics.waitFor();
  await semantics.evaluate(element => element.click());
  await expect(page.getByText('Nuestros productos', { exact: true })).toBeVisible();
}

test('medium viewport keeps the quantity dialog usable when available height shrinks', async ({ page }) => {
  await page.setViewportSize({ width: 768, height: 1024 });
  await openShop(page);
  await page.getByRole('button', { name: 'Cantidad de SKU-001: 0', exact: true }).click();
  // Browser resize exercises the same reduced space; Flutter tests supply real viewInsets.
  await page.setViewportSize({ width: 768, height: 460 });
  await page.getByRole('textbox').last().fill('18');
  const apply = page.getByRole('button', { name: 'Aplicar', exact: true });
  await expect(apply).toBeInViewport({ ratio: 1 });
  await apply.click();
  await expect(page.getByRole('button', { name: 'Cantidad de SKU-001: 18', exact: true })).toBeVisible();
});

test('real quote, gifts, preserved snapshot and safe replay after lost response + reload', async ({ page }, testInfo) => {
  const pageErrors = [];
  page.on('pageerror', error => pageErrors.push(error.message));
  await openShop(page);
  await page.screenshot({ path: testInfo.outputPath('catalogue-desktop.png') });
  await page.getByRole('button', { name: 'Por volumen · 12 uds.', exact: true }).click();
  await page.getByText('Contado', { exact: true }).click();
  const quoted = page.waitForResponse(r => r.url().endsWith('/api/quotes') && r.request().method() === 'POST');
  await page.getByRole('button', { name: 'Cotizar pedido', exact: true }).click();
  const quote = await (await quoted).json();
  expect(quote.total).toEqual({ amount: '1363', currency: 'PEN', scale: 2 });
  expect(quote.discountTotal.amount).toBe('45');
  expect(quote.gifts[0].quantity).toBe(2);
  await expect(page.getByText('S/ 13,63', { exact: true })).toBeVisible();
  await expect(page.getByText('2 × Obsequio de la casa', { exact: true })).toBeVisible();
  await page.screenshot({ path: testInfo.outputPath('quote-desktop.png') });

  let originalKey;
  let confirmed;
  await page.route('**/api/orders', async route => {
    originalKey = route.request().headers()['idempotency-key'];
    const response = await route.fetch(); // Commit the real order, then lose only the browser response.
    expect(response.status()).toBe(201);
    confirmed = await response.json();
    await route.abort('failed');
  });
  await page.getByRole('button', { name: 'Confirmar pedido', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Reintentar confirmación', exact: true })).toBeEnabled();
  expect(confirmed.total).toEqual(quote.total);
  await page.unroute('**/api/orders');
  await page.reload();
  const semantics = page.locator('flt-semantics-placeholder');
  if (await semantics.count()) await semantics.evaluate(element => element.click());
  const replay = page.waitForResponse(r => r.url().endsWith('/api/orders') && r.request().method() === 'POST');
  await page.getByRole('button', { name: 'Reintentar confirmación', exact: true }).click();
  const response = await replay;
  expect(response.request().headers()['idempotency-key']).toBe(originalKey);
  expect((await response.json()).orderId).toBe(confirmed.orderId);
  await expect(page.getByText('PEDIDO CONFIRMADO', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Ver mis pedidos', exact: true }).click();
  await expect(page.getByText(confirmed.orderId, { exact: true })).toBeVisible();
  const read = page.waitForResponse(r => r.url().endsWith(`/api/orders/${confirmed.orderId}`));
  await page.getByRole('button', { name: 'Consultar de nuevo', exact: true }).click();
  expect((await (await read).json()).total).toEqual(quote.total);
  await page.getByRole('textbox', { name: 'Identificador del pedido' }).fill(confirmed.orderId);
  const lookup = page.waitForResponse(r => r.url().endsWith(`/api/orders/${confirmed.orderId}`));
  await page.getByRole('button', { name: 'Consultar pedido', exact: true }).click();
  expect((await lookup).status()).toBe(200);
  await page.screenshot({ path: testInfo.outputPath('order-desktop.png') });
  expect(pageErrors).toEqual([]);
});

test('real combo applies before volume discount and taxes', async ({ page }) => {
  await openShop(page);
  await page.getByRole('button', { name: 'Combo + volumen', exact: true }).click();
  const response = page.waitForResponse(r => r.url().endsWith('/api/quotes'));
  await page.getByRole('button', { name: 'Cotizar pedido', exact: true }).click();
  const q = await (await response).json();
  expect(q.total.amount).toBe('1056');
  expect(q.discountTotal.amount).toBe('105');
  expect(q.taxableBase.amount).toBe('895');
  expect(q.taxTotal.amount).toBe('161');
  expect(q.adjustments.filter(a => a.promotionType === 'SCALE').reduce((sum, a) => sum + a.quantity, 0)).toBe(1);
  await expect(page.getByText('S/ 10,56', { exact: true })).toBeVisible();
});

test('mobile country change, CLP quote and insufficient credit cannot confirm', async ({ page }, testInfo) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await openShop(page);
  await page.screenshot({ path: testInfo.outputPath('catalogue-mobile.png') });
  await page.getByText('Perú · PEN', { exact: true }).click();
  await page.getByRole('menuitem', { name: 'Chile · CLP', exact: true }).click();
  await page.getByRole('button', { name: 'Por volumen · 12 uds.', exact: true }).click();
  await page.getByRole('button', { name: /Ir a tu pedido/ }).click();
  let response = page.waitForResponse(r => r.url().endsWith('/api/quotes'));
  await page.getByRole('button', { name: 'Cotizar pedido', exact: true }).click();
  let q = await (await response).json();
  expect(q.country).toBe('CL'); expect(q.total).toEqual({ amount: '13745', currency: 'CLP', scale: 0 });
  await expect(page.getByText('$ 13.745', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: /Ir a tu pedido/ }).click();
  await page.screenshot({ path: testInfo.outputPath('quote-mobile.png') });
  // Quantity control opens a validated editor, including large quantities for credit boundaries.
  await page.getByRole('button', { name: 'Cantidad de SKU-001: 12', exact: true }).click();
  await page.getByRole('textbox').last().fill('1000000');
  await page.getByRole('button', { name: 'Aplicar', exact: true }).click();
  await page.getByRole('button', { name: /Ir a tu pedido/ }).click();
  response = page.waitForResponse(r => r.url().endsWith('/api/quotes'));
  await page.getByRole('button', { name: 'Cotizar pedido', exact: true }).click();
  q = await (await response).json();
  expect(q.creditEvaluation.eligible).toBe(false);
  await expect(page.getByText('Crédito insuficiente', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Confirmar pedido', exact: true })).toBeDisabled();
});
