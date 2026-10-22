// The brand's colours, read out of the sources that hold them: no browser, no
// Playwright, no app. Run with Node's test runner; see README.md.
//
// The fork's green disc wore two greens; the yellow disc wears the brand's
// own yellow instead. This says the yellow is in every source that paints
// the mark, and that the old green is in none of them: a half-green mark
// would be a source left behind.
//
//   node --test internal/gui/tests/brand-colors.test.cjs
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const { test } = require("node:test");

const repoRoot = path.resolve(__dirname, "../../..");

// the greens the green disc wore, in every spelling the sources use
const OLD_GREEN = ["14b789", "2ad39b", "0x14, 0xb7, 0x89", "0x2a, 0xd3, 0x9b"];

// what each brand source must hold: the brand's yellow, light and dark
const BRAND = {
  light: "#FACC15",
  dark: "#FDE047",
};

function read(rel) {
  return fs.readFileSync(path.join(repoRoot, rel), "utf8");
}

// the sources that paint the mark, and what each must say
const sources = [
  {
    file: "build/icon/gen.go",
    // the generator's own bytes: the light one and the dark theme's shade
    light: ["0xfa, 0xcc, 0x15"],
    dark: ["0xfd, 0xe0, 0x47"],
  },
  {
    file: "internal/gui/assets/app.css",
    light: ["--logo-fill: #facc15"],
    dark: ["--logo-fill: #fde047"],
    // the business greens are states, not the brand: they must stay
    keep: ["--green: #189a58", "--green-soft: #e4f6ec"],
  },
  {
    file: "internal/gui/assets/icons/favicon.svg",
    light: ["fill:#FACC15"],
    dark: ["fill:#FDE047"],
  },
  {
    file: "internal/provider/magpie.svg",
    light: ["fill:#FACC15"],
    dark: ["fill:#FDE047"],
  },
  {
    file: "internal/provider/signin.go",
    light: ["fill:#FACC15"],
    dark: ["fill:#FDE047"],
  },
  {
    file: "build/darwin/AppIcon.icon/icon.json",
    light: ["srgb:0.98039,0.80000,0.08235"],
    dark: ["srgb:0.99216,0.87843,0.27843"],
  },
];

test("every brand source paints the mark the brand's yellow", () => {
  for (const s of sources) {
    const text = read(s.file);
    const lower = text.toLowerCase();
    for (const want of s.light) {
      assert.ok(text.includes(want), `${s.file}: the light colour ${want} is missing`);
    }
    for (const want of s.dark) {
      assert.ok(text.includes(want), `${s.file}: the dark colour ${want} is missing`);
    }
    for (const keep of s.keep || []) {
      assert.ok(text.includes(keep), `${s.file}: ${keep} is gone: the state colours are not the brand's`);
    }
    for (const old of OLD_GREEN) {
      assert.ok(!lower.includes(old.toLowerCase()), `${s.file}: the old green ${old} is still there`);
    }
  }
});

test("the yellow is the one the brand says it is", () => {
  // the generator's bytes are the yellow, not another colour in the same place
  const gen = read("build/icon/gen.go");
  const light = gen.match(/gr,\s*gg,\s*gb\s*:=\s*([^;\n]+)/);
  const dark = gen.match(/gr,\s*gg,\s*gb\s*=\s*([^;\n]+)/);
  assert.ok(light && dark, "build/icon/gen.go: the two colours are read");
  const bytes = (s) => (s.match(/0x[0-9a-fA-F]{2}/g) || []).map((h) => parseInt(h.slice(2), 16));
  assert.deepEqual(bytes(light[1]), [0xfa, 0xcc, 0x15], "build/icon/gen.go: the light colour");
  assert.deepEqual(bytes(dark[1]), [0xfd, 0xe0, 0x47], "build/icon/gen.go: the dark colour");
  // and the CSS the page paints the mark with says the same
  const css = read("internal/gui/assets/app.css");
  assert.ok(css.includes(`--logo-fill: ${BRAND.light.toLowerCase()};`), "app.css: the light --logo-fill");
  assert.ok(css.includes(`--logo-fill: ${BRAND.dark.toLowerCase()};`), "app.css: the dark --logo-fill");
});
