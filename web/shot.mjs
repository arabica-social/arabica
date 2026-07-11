import { chromium } from "@playwright/test";

const BASE = "http://localhost:18080";
const CHROMIUM =
  "/nix/store/828nflkpw300plz3n4f6c3axfirkm6qn-chromium-149.0.7827.196/bin/chromium";

const browser = await chromium.launch({ executablePath: CHROMIUM });
const ctx = await browser.newContext({
  viewport: { width: 1280, height: 900 },
  deviceScaleFactor: 2,
});
const page = await ctx.newPage();
await page.goto(`${BASE}/`, { waitUntil: "networkidle" });
await page.waitForTimeout(1500);
await page.screenshot({ path: `/tmp/shot-hero-bean.png`, fullPage: false });
console.log("shot hero");
await browser.close();
