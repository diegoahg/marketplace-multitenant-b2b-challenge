import { chromium } from '@playwright/test';
import { mkdir } from 'node:fs/promises';
import { fileURLToPath, pathToFileURL } from 'node:url';
import path from 'node:path';

const root = fileURLToPath(new URL('../../', import.meta.url));
const captures = path.join(root, 'frontend/test-results/adr');
await mkdir(captures, { recursive: true });
const browser = await chromium.launch({ channel: 'chrome', headless: true });
try {
  const page = await browser.newPage();
  await page.goto(pathToFileURL(path.join(root, 'Documentation/adr.html')).href);
  for (const width of [390, 768, 1200]) {
    await page.setViewportSize({ width, height: 900 });
    const overflow = await page.evaluate(() =>
      document.documentElement.scrollWidth > window.innerWidth);
    if (overflow) throw new Error(`Horizontal overflow at ${width}px`);
    await page.screenshot({ path: path.join(captures, `${width}.png`), fullPage: true });
  }
  const pdf = await page.pdf({
    path: path.join(root, 'Documentation/ADR.pdf'),
    preferCSSPageSize: true,
    printBackground: true,
  });
  const pages = [...pdf.toString('latin1').matchAll(/\/Type\s*\/Page\b/g)].length;
  if (pages !== 1) throw new Error(`Expected one A4 page; got ${pages}`);
  console.log('ADR: one A4 page; no horizontal overflow at 390, 768 or 1200px.');
} finally {
  await browser.close();
}
