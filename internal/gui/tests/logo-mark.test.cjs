// Run with Node's test runner and Playwright on the module path; see README.md.
// The header's mark: the app's disc (build/icon/magpie.svg) in the app's yellow
// (build/icon/gen.go), breathing when the window opens and the pointer comes
// over it.
// The shape is read out of the brand source (build/icon/magpie.svg), so a
// change to the disc holds the mark to it; the colours are fixed expectations
// written out here, so the mark is held to the brand's own yellow and not to
// whatever the generator happens to paint.
// The old shape is out of the mark; the Routing page's bird path stays in the
// document, because that page still draws its marks with a use of #bird
// (plugins/routing/client.js).
const assert = require("node:assert/strict");
const fs = require("node:fs/promises");
const path = require("node:path");
const { test } = require("node:test");
const { chromium, webkit } = require("playwright");

// The brand's own colours, written out: the light theme's yellow and the
// dark theme's own shade of it. A test that read them from the implementation
// would hold whatever the implementation says.
const BRAND = {
  light: [250, 204, 21], // #FACC15
  dark: [253, 224, 71], // #FDE047
};

const repoRoot = path.resolve(__dirname, "../../..");
const assets = process.env.MAGPIE_LOGO_ASSETS || path.resolve(__dirname, "../assets");
// the brand source the mark is held to: the shape build/icon/magpie.svg draws
// and the colours build/icon/gen.go fills it with
const brandSVG = process.env.MAGPIE_LOGO_SVG || path.join(repoRoot, "build", "icon", "magpie.svg");
const brandGen = process.env.MAGPIE_LOGO_GEN || path.join(repoRoot, "build", "icon", "gen.go");

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

// a file that can't be read says nothing about the brand
async function readText(file) {
  try {
    return await fs.readFile(file, "utf8");
  } catch {
    return null;
  }
}

// the disc the brand source draws: the points on the path's curve, a
// command's end with its control points left out (a circle's control points
// sit off it, on its tangents). Those are the disc's own points, so the
// centre is the middle of their box and the radius how far they are from it
function discOf(svg) {
  const box = svg.match(/viewBox="([^"]+)"/);
  const paths = svg.match(/<path\b/g) || [];
  const d = svg.match(/\sd="([^"]+)"/);
  if (!box || !d || paths.length !== 1) return null;
  const points = pathPoints(d[1]);
  if (!points.length) return null;
  const xs = points.map((p) => p[0]);
  const ys = points.map((p) => p[1]);
  const cx = (Math.min(...xs) + Math.max(...xs)) / 2;
  const cy = (Math.min(...ys) + Math.max(...ys)) / 2;
  const r = points.reduce((sum, p) => sum + Math.hypot(p[0] - cx, p[1] - cy), 0) / points.length;
  return { viewBox: box[1].trim().split(/\s+/).join(" "), cx, cy, r };
}

// the points on the curve: the end of every move, line and curve
function pathPoints(d) {
  const tokens = d.match(/[A-Za-z]|-?\d*\.?\d+(?:[eE][-+]?\d+)?/g) || [];
  const points = [];
  let i = 0;
  let cmd = "";
  const num = () => Number(tokens[i++]);
  while (i < tokens.length) {
    if (/[A-Za-z]/.test(tokens[i])) cmd = tokens[i++];
    switch (cmd) {
      case "M":
      case "L": {
        const x = num();
        const y = num();
        if (!Number.isFinite(x) || !Number.isFinite(y)) return null;
        points.push([x, y]);
        cmd = "L"; // the pairs after a move are lines
        break;
      }
      case "C": {
        num();
        num();
        num();
        num(); // the two control points, off the curve
        const x = num();
        const y = num();
        if (!Number.isFinite(x) || !Number.isFinite(y)) return null;
        points.push([x, y]);
        break;
      }
      case "Z":
        break;
      default:
        return null; // a command this does not read
    }
  }
  return points;
}

// the two colours the brand source paints the icon with: the light one and the
// dark theme's own shade of it, read out of the generator's own source
function brandOf(gen) {
  const start = gen.indexOf("func app(");
  if (start < 0) return null;
  const end = gen.indexOf("\nfunc ", start);
  const body = gen.slice(start, end < 0 ? undefined : end);
  const light = body.match(/gr,\s*gg,\s*gb\s*:=\s*([^;\n]+)/);
  const dark = body.match(/gr,\s*gg,\s*gb\s*=\s*([^;\n]+)/);
  if (!light || !dark) return null;
  const shade = body.indexOf("if dark");
  if (shade < 0 || body.indexOf(dark[0]) < shade) return null;
  const bytes = (s) => (s.match(/0x[0-9a-fA-F]{2}/g) || []).map((h) => parseInt(h.slice(2), 16));
  const l = bytes(light[1]);
  const d = bytes(dark[1]);
  if (l.length !== 3 || d.length !== 3) return null;
  return { light: l, dark: d };
}

// the channels a fill names, its three colours
function channels(fill) {
  const m = String(fill).match(/rgba?\(([^)]+)\)/);
  if (!m) return null;
  const parts = m[1].split(/[,/]/).map((v) => parseFloat(v));
  return parts.length >= 3 ? parts.slice(0, 3) : null;
}

for (const engine of (process.env.BROWSER ? [process.env.BROWSER] : ["chromium", "webkit"])) {
  test(`${engine}: the header's mark is the app's disc`, async (t) => {
    const browser = await (engine === "webkit" ? webkit.launch() : chromium.launch({ channel: "chromium" }));
    t.after(() => browser.close());
    for (const theme of ["light", "dark"]) {
      const context = await browser.newContext({ viewport: { width: 860, height: 700 }, colorScheme: theme });
      const page = await context.newPage();
      page.setDefaultTimeout(5000);
      // the page's own script is a megabyte and the suite runs many pages at
      // once, so the navigation gets the room Playwright gives it by default
      // (30s) and only the waits and the assertions keep the 5s
      page.setDefaultNavigationTimeout(30000);
      const errors = [];
      page.on("pageerror", (e) => errors.push(e.message));
      await page.route("**/*", serve(theme));
      await page.goto("http://magpie.test/");
      await page.waitForSelector("#nav button.on");
      const where = `the ${theme} theme`;
      const logo = page.locator(".brand .logo");
      assert.ok(await logo.isVisible(), `${where}: the mark`);
      const svg = logo.locator("svg");
      assert.equal(await svg.getAttribute("width"), "18", `${where}: the mark's width`);
      assert.equal(await svg.getAttribute("height"), "18", `${where}: the mark's height`);
      // the disc the app icon is drawn: the same circle, nothing else in it
      const circle = logo.locator("circle");
      assert.equal(await circle.count(), 1, `${where}: the mark is the disc`);
      assert.equal(await logo.locator("path, use, g").count(), 0, `${where}: the old shape is out of the mark`);
      assert.equal(await logo.locator("[id]").count(), 0, `${where}: the mark holds no definition`);
      // and the disc it draws is the brand's own: the shape and the colours are
      // read out of the brand source, so a change there holds the mark to it
      const disc = discOf(await readText(brandSVG));
      assert.ok(disc, "the brand source: build/icon/magpie.svg's disc is read");
      assert.equal(await svg.getAttribute("viewBox"), disc.viewBox, `${where}: the disc's grid`);
      for (const k of ["cx", "cy", "r"]) {
        const got = Number(await circle.getAttribute(k));
        assert.ok(Math.abs(got - disc[k]) < 1e-6, `${where}: the disc's ${k} is ${got}, want the brand's ${disc[k]}`);
      }
      // the app's yellow, the shade of it the theme asks for
      const brand = brandOf(await readText(brandGen));
      assert.ok(brand, "the brand source: build/icon/gen.go's colours are read");
      // and they are the brand's own yellow, written out here: the generator
      // is held to the yellow, not the other way round
      assert.deepEqual(brand.light, BRAND.light, "the brand source's light colour is the brand's yellow");
      assert.deepEqual(brand.dark, BRAND.dark, "the brand source's dark colour is the brand's own shade");
      const fill = await circle.evaluate((c) => getComputedStyle(c).fill);
      assert.deepEqual(channels(fill), BRAND[theme], `${where}: the mark's yellow`);
      // the wordmark keeps the ink, only the mark is yellow
      const word = page.locator(".brand > span:not(.logo)");
      assert.equal((await word.textContent()).trim(), "magpie", `${where}: the wordmark`);
      const ink = await word.evaluate((w) => getComputedStyle(w).color);
      assert.equal(ink, await page.evaluate(() => getComputedStyle(document.body).color), `${where}: the wordmark keeps the ink`);
      assert.notDeepEqual(channels(ink), BRAND[theme], `${where}: the wordmark is not the mark's yellow`);
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
    page.setDefaultNavigationTimeout(30000);
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
