// The agent mascot: a pixel creature in a wizard hat, drawn as SVG.
// Shared by the simulator (index.html) and the animation gallery (gallery.html);
// its animations are in mascot.css.

const COLORS = ["#D85A30", "#1D9E75", "#7F77DD", "#EF9F27", "#378ADD", "#D4537E", "#97C459", "#5DCAA5"];
const EYE = "#121211";
// X body, O eye, - closed eye, A antenna. Four body shapes, taken in turn by mascot index.
const SHAPES = [
  { // round with little legs
    awake: ["..XXXXXX..", ".XXXXXXXX.", "XXOXXXXOXX", "XXXXXXXXXX", ".XXXXXXXX.", ".X.X..X.X."],
    sleep: ["..XXXXXX..", ".XXXXXXXX.", "XX--XX--XX", "XXXXXXXXXX", ".XXXXXXXX.", ".XXXXXXXX."] },
  { // antennas
    awake: ["A........A", ".XXXXXXXX.", "XXOOXXOOXX", "XXXXXXXXXX", "XXXXXXXXXX", "X.X....X.X"],
    sleep: ["A........A", ".XXXXXXXX.", "XX--XX--XX", "XXXXXXXXXX", "XXXXXXXXXX", "XXXXXXXXXX"] },
  { // ghost with a wavy hem
    awake: ["..XXXXXX..", ".XXXXXXXX.", ".XOXXXXOX.", ".XXXXXXXX.", ".XXXXXXXX.", ".X.XX.XX.X"],
    sleep: ["..XXXXXX..", ".XXXXXXXX.", ".--XXXX--.", ".XXXXXXXX.", ".XXXXXXXX.", ".X.XX.XX.X"] },
  { // slime with a drop on top
    awake: ["....XX....", "..XXXXXX..", ".XXOXXOXX.", "XXXXXXXXXX", "XXXXXXXXXX", ".XXXXXXXX."],
    sleep: ["....XX....", "..XXXXXX..", ".X--XX--X.", "XXXXXXXXXX", "XXXXXXXXXX", ".XXXXXXXX."] },
];

// The mascot wears a wizard hat and holds a wand. The canvas is 13x10 cells:
// the hat on top, the body below it, the wand to the right of the body.
// H hat, Y gold, B wood, * sparkle.
const HAT = [".....H.......", "....HHH......", "...HHYHH.....", ".HHHHHHHHH..."];
const WAND = [[10, 7, "B"], [11, 6, "B"], [12, 5, "Y"]];
// While working, sparkles fly off the wand in two alternating frames.
const SPARKLES = [[[11, 3], [12, 1]], [[12, 3], [10, 2], [11, 0]]];
const MAGIC = { H: "#534AB7", Y: "#F2C14E", B: "#8A5A2B", "*": "#7FD6FF" };
// One snore (breath in and out) takes this long; the Zzz rise to the same rhythm.
const SNORE = 3000;

function mascotSVG(index, status) {
  const color = COLORS[index % COLORS.length];
  const shape = SHAPES[index % SHAPES.length];
  const rows = status === "done" || status === "idle" ? shape.sleep : shape.awake;
  const u = 6;
  const cell = (x, y, fill) => `<rect x="${x*u}" y="${y*u}" width="${u}" height="${u}" fill="${fill}"/>`;
  // On shapes with something sticking out on top (antennas, the slime's drop)
  // the hat sits one row lower and covers it.
  const hatTop = rows[0].replace(/\./g, "").length < 4 ? 1 : 0;
  const bodyTop = 4;
  // Animations are timed by the wall clock (a negative delay), so redrawing the card doesn't restart them.
  const clock = period => -(Date.now() % period);
  const asleep = status === "done" || status === "idle";

  let body = "";
  rows.forEach((row, y) => [...row].forEach((ch, x) => {
    const py = bodyTop + y;
    if ((ch === "X" || ch === "A") && py >= hatTop + HAT.length) body += cell(x, py, color);
    else if (ch === "O") body += cell(x, py, EYE);
    // A closed eye is a thin lid line on the body color, so it doesn't read as an open eye.
    else if (ch === "-") body += cell(x, py, color) + `<rect x="${x*u}" y="${py*u+u/2}" width="${u}" height="${u/3}" fill="${EYE}"/>`;
  }));
  // The purple mascot would blend into the purple hat, so it gets a burgundy one.
  const hat = { ...MAGIC, H: color === "#7F77DD" ? "#993556" : MAGIC.H };
  HAT.forEach((row, y) => [...row].forEach((ch, x) => { if (ch !== ".") body += cell(x, hatTop + y, hat[ch]); }));

  // A sleeping mascot snores: the body and hat swell and sink with each breath.
  let out = asleep ? `<g class="snore" style="animation-delay:${clock(SNORE)}ms">${body}</g>` : body;
  WAND.forEach(([x, y, ch]) => { out += cell(x, y, MAGIC[ch]); });
  if (status === "working") {
    const period = 700;
    out += SPARKLES.map((pts, i) =>
      `<g class="fr" style="animation-delay:${clock(period) - (period / 2) * i}ms">${pts.map(([x, y]) => cell(x, y, MAGIC["*"])).join("")}</g>`).join("");
  }
  if (asleep) {
    // Three Zs rise from the hat one after another, one per breath, growing and fading out.
    out += [0, 1, 2].map(i =>
      `<text class="zzz" x="58" y="26" font-size="11" font-weight="700" fill="#b4b2a9" font-family="monospace" style="animation-delay:${clock(SNORE) - (SNORE / 3) * i}ms">Z</text>`).join("");
  }
  return `<svg class="mascot" width="62" height="48" viewBox="0 0 78 60" shape-rendering="crispEdges" aria-hidden="true">${out}</svg>`;
}
