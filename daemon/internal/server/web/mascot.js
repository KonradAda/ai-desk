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
// One hop while working takes this long, and the wand sparkles switch frames
// on the way up and on the way down. A pulse while waiting takes PULSE.
// Keep in step with bob and pulse in mascot.css.
const HOP = 800;
const PULSE = 1100;

// Changes between sleeping and awake play a short animation of eye frames
// instead of jumping straight to the new state. Each lasts TRANSITION ms;
// the frame timings are in mascot.css (wk0-wk6 waking up, fa0-fa6 falling asleep).
const TRANSITION = 1800;
const TRANSITIONS = {
  // asleep, eyes half open, open, two blinks, then wide awake with a little hop
  wake: { cls: "wk", eyes: ["closed", "half", "open", "closed", "open", "closed", "open"] },
  // eyes droop, it fights it once, droops again, closes, one last peek, asleep
  sleep: { cls: "fa", eyes: ["open", "half", "open", "half", "closed", "half", "closed"] },
};

// mascotSVG draws the mascot for an agent. change, when set, says the agent
// went from asleep to awake (kind "wake") or back ("sleep") ago ms ago; for
// the first TRANSITION ms the mascot plays that change instead of its status
// animation.
function mascotSVG(index, status, change = null) {
  const color = COLORS[index % COLORS.length];
  const shape = SHAPES[index % SHAPES.length];
  const u = 6;
  const cell = (x, y, fill) => `<rect x="${x*u}" y="${y*u}" width="${u}" height="${u}" fill="${fill}"/>`;
  // On shapes with something sticking out on top (antennas, the slime's drop)
  // the hat sits one row lower and covers it.
  const hatTop = shape.awake[0].replace(/\./g, "").length < 4 ? 1 : 0;
  const bodyTop = 4;
  // Animations are timed by a clock (a negative delay), so redrawing the card
  // doesn't restart them. After a change the clock starts when the change
  // ends, so the hop, breathing and Zzz begin at the start of their cycle,
  // right where the change left the mascot; otherwise it's the wall clock.
  const since = change ? Math.max(0, change.ago - TRANSITION) : Date.now();
  const clock = period => -(since % period);
  const asleep = status === "done" || status === "idle";
  const moving = change && change.ago < TRANSITION ? TRANSITIONS[change.kind] : null;
  // The purple mascot would blend into the purple hat, so it gets a burgundy one.
  const hat = { ...MAGIC, H: color === "#7F77DD" ? "#993556" : MAGIC.H };

  // drawBody draws body and hat; halfOpen turns open eyes into sleepy half-open ones.
  const drawBody = (rows, halfOpen = false) => {
    let body = "";
    rows.forEach((row, y) => [...row].forEach((ch, x) => {
      const py = bodyTop + y;
      if ((ch === "X" || ch === "A") && py >= hatTop + HAT.length) body += cell(x, py, color);
      else if (ch === "O" && halfOpen) body += cell(x, py, color) + `<rect x="${x*u}" y="${py*u+u/2}" width="${u}" height="${u/2}" fill="${EYE}"/>`;
      else if (ch === "O") body += cell(x, py, EYE);
      // A closed eye is a thin lid line on the body color, so it doesn't read as an open eye.
      else if (ch === "-") body += cell(x, py, color) + `<rect x="${x*u}" y="${py*u+u/2}" width="${u}" height="${u/3}" fill="${EYE}"/>`;
    }));
    HAT.forEach((row, y) => [...row].forEach((ch, x) => { if (ch !== ".") body += cell(x, hatTop + y, hat[ch]); }));
    return body;
  };

  let out;
  if (moving) {
    const eyes = { closed: drawBody(shape.sleep), half: drawBody(shape.awake, true), open: drawBody(shape.awake) };
    out = moving.eyes.map((e, i) =>
      `<g class="tr ${moving.cls}${i}" style="animation-delay:${-change.ago}ms">${eyes[e]}</g>`).join("");
  } else if (asleep) {
    // A sleeping mascot snores: the body and hat swell and sink with each breath.
    out = `<g class="snore" style="animation-delay:${clock(SNORE)}ms">${drawBody(shape.sleep)}</g>`;
  } else {
    out = drawBody(shape.awake);
  }
  WAND.forEach(([x, y, ch]) => { out += cell(x, y, MAGIC[ch]); });
  if (moving) return `<svg class="mascot changing" width="62" height="48" viewBox="0 0 78 60" shape-rendering="crispEdges" aria-hidden="true">${out}</svg>`;
  if (status === "working") {
    const period = HOP;
    out += SPARKLES.map((pts, i) =>
      `<g class="fr" style="animation-delay:${clock(period) - (period / 2) * i}ms">${pts.map(([x, y]) => cell(x, y, MAGIC["*"])).join("")}</g>`).join("");
  }
  if (asleep) {
    // Three Zs rise from the hat one after another, one per breath, growing and fading out.
    out += [0, 1, 2].map(i =>
      `<text class="zzz" x="58" y="26" font-size="11" font-weight="700" fill="#b4b2a9" font-family="monospace" style="animation-delay:${clock(SNORE) - (SNORE / 3) * i}ms">Z</text>`).join("");
  }
  // The hop and the pulse run on the whole mascot; timing them by the clock too
  // keeps them smooth when the card is redrawn.
  const whole = status === "working" ? clock(HOP) : status === "waiting" ? clock(PULSE) : null;
  const style = whole === null ? "" : ` style="animation-delay:${whole}ms"`;
  return `<svg class="mascot" width="62" height="48" viewBox="0 0 78 60" shape-rendering="crispEdges" aria-hidden="true"${style}>${out}</svg>`;
}
