// Run with Node's test runner and Playwright on the module path; see README.md.
// The header's mark: the app's disc (build/icon/magpie.svg) in the app's green,
// breathing when the window opens and the pointer comes over it. The bird is
// out of the mark; its path stays in the document, because the Routing page
// still draws its marks with a use of #bird (plugins/routing/client.js).
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

const assets = process.env.MAGPIE_LOGO_ASSETS || path.resolve(__dirname, "../assets");

// the disc build/icon/magpie.svg draws, and the two greens build/icon/gen.go
// fills it with
const DISC = { cx: "22", cy: "22", r: "21.6" };
const GREEN = { light: "rgb(20, 183, 137)", dark: "rgb(42, 211, 155)" };

function serve(theme) {
  const providers = { providers: [], presets: [], excluded: [], gateway: { running: true, window: true } };
  const state = { agents: [], profiles: [], settings: { lang: "en", theme } };
  return async (route) => {
    const url = new URL(route.request().url());
    const json = (data) => route.fulfill({ json: data });
    if (url.pathname === "/boot.js") return route.fulfill({ contentType: "text/javascript", body: `window.bootPrefs = {lang:"en",theme:"${theme}",web:true};` });
    if (url.pathname === "/wails/runtime.js") return route.fulfill({ contentType: "text/javascript", body: "export const Window = {};" });
    if (url.pathname === "/api/state") return json(state);
    if (url.pathname === "/api/providers") return json(providers);
    if (url.pathname === "/api/groups") return json({ groups: [] });
    if (url.pathname.startsWith("/api/")) return json({});
    const file = path.join(assets, url.pathname === "/" ? "index.html" : url.pathname);
    const contentType = { ".html": "text/html", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml", ".png": "image/png" }[path.extname(file)];
    await route.fulfill({ body: await fs.readFile(file), contentType });
  };
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: the header's mark is the app's disc`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const theme of ["light", "dark"]) {
      const context = await browser.newContext({ viewport: { width: 860, height: 700 }, colorScheme: theme });
      const page = await context.newPage();
      page.setDefaultTimeout(5000);
      const errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(theme));
      await page.goto("http://magpie.test/");
      await page.waitForSelector("#nav button.on");
      const where = `the ${theme} theme`;
      const logo = page.locator(".brand .logo");
      assert.ok(await logo.isVisible(), `${where}: the mark`);
      const svg = logo.locator("svg");
      assert.equal(await svg.getAttribute("viewBox"), "0 0 44 44", `${where}: the disc's grid`);
      assert.equal(await svg.getAttribute("width"), "18", `${where}: the mark's width`);
      assert.equal(await svg.getAttribute("height"), "18", `${where}: the mark's height`);
      // the disc the app icon is drawn: the same circle on the same grid
      const circle = logo.locator("circle");
      assert.equal(await circle.count(), 1, `${where}: the mark is the disc`);
      for (const [k, v] of Object.entries(DISC)) assert.equal(await circle.getAttribute(k), v, `${where}: the disc's ${k}`);
      // the app's green, the shade of it the theme asks for
      assert.equal(await circle.evaluate((c) => getComputedStyle(c).fill), GREEN[theme], `${where}: the mark's green`);
      // the bird is out of the mark: nothing in it draws the old shape
      assert.equal(await logo.locator("path, use, g").count(), 0, `${where}: the bird is out of the mark`);
      assert.equal(await logo.locator("[id]").count(), 0, `${where}: the mark holds no definition`);
      // the wordmark keeps the ink, only the mark is green
      const word = page.locator(".brand > span:not(.logo)");
      assert.equal((await word.textContent()).trim(), "magpie", `${where}: the wordmark`);
      const ink = await word.evaluate((w) => getComputedStyle(w).color);
      assert.equal(ink, await page.evaluate(() => getComputedStyle(document.body).color), `${where}: the wordmark keeps the ink`);
      assert.notEqual(ink, GREEN[theme], `${where}: the wordmark is not the mark's green`);
      assert.deepEqual(errors, [], where);
      await context.close();
    }
  });

  test(`${engine}: the mark breathes when the window opens and on hover`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const context = await browser.newContext({ viewport: { width: 860, height: 700 } });
    const page = await context.newPage();
    page.setDefaultTimeout(10000);
    const errors = [];
    page.on("pageerror", (e) => errors.push(e.message));
    await page.route("**/*", serve("light"));
    await page.goto("http://magpie.test/");
    await page.waitForSelector("#nav button.on");
    const mark = page.locator(".brand .logo .mark");
    assert.equal(await mark.count(), 1, "the mark is the disc, and it breathes");
    // the window opens and the mark breathes
    await page.waitForFunction(() => document.querySelector(".brand .logo")?.classList.contains("wag"));
    const breath = await mark.evaluate((m) => {
      const s = getComputedStyle(m);
      return { name: s.animationName, dur: s.animationDuration };
    });
    assert.equal(breath.name, "logo-pulse", "the mark breathes");
    assert.equal(breath.dur, "0.9s", "the breath takes the tray flap's .9s");
    // and it rests when the breath is over
    await page.waitForFunction(() => !document.querySelector(".brand .logo")?.classList.contains("wag"));
    // the pointer over the mark breathes it again
    await page.locator(".brand").hover();
    await page.waitForFunction(() => document.querySelector(".brand .logo")?.classList.contains("wag"));
    assert.deepEqual(errors, [], "the mark breathes without a page error");
    await context.close();
  });

  test(`${engine}: the Routing page's bird still has its shape to draw`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    const context = await browser.newContext({ viewport: { width: 860, height: 700 } });
    const page = await context.newPage();
    page.setDefaultTimeout(5000);
    await page.route("**/*", serve("light"));
    await page.goto("http://magpie.test/");
    await page.waitForSelector("#nav button.on");
    // the Routing page draws its marks with a use of #bird: the path has to
    // stay in the document, whatever the mark draws now
    const bird = await page.evaluate(() => {
      const def = document.getElementById("bird");
      const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
      svg.setAttribute("viewBox", "0 0 44 44");
      svg.innerHTML = '<use href="#bird"/>';
      document.body.append(svg);
      const box = svg.getBBox();
      svg.remove();
      return { tag: def?.tagName, w: box.width, h: box.height };
    });
    assert.equal(bird.tag, "path", "the bird's path is still in the document");
    assert.ok(bird.w > 0 && bird.h > 0, "the bird still draws: " + JSON.stringify(bird));
    await context.close();
  });
}
