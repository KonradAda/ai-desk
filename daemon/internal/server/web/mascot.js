// The agent mascot: a creature in a wizard hat holding a wand, drawn as SVG.
// Shared by the simulator (index.html) and the animation gallery (gallery.html);
// its animations are in mascot.css.

const COLORS = ["#D85A30", "#1D9E75", "#7F77DD", "#EF9F27", "#378ADD", "#D4537E", "#97C459", "#5DCAA5"];
// Each mascot's hat, one per body color above: deep shades that stand apart
// from the body and from each other, all dark enough for the gold star.
const HATS = ["#534AB7", "#993556", "#1E6F6A", "#2C3E8C", "#8E2A6B", "#2E6B3A", "#5B2C83", "#9A3B2E"];
const EYE = "#121211";
// Y gold, B wood, * sparkle.
const MAGIC = { Y: "#F2C14E", B: "#8A5A2B", "*": "#7FD6FF" };
// One snore (breath in and out) takes this long; the Zzz rise to the same rhythm.
const SNORE = 3000;
// One hop while working or waiting takes this long, and the wand sparkles
// switch frames on the way up and on the way down. Keep in step with bob in mascot.css.
const HOP = 800;
// While waiting for the user the mascot holds up a sign and waves it once
// every WAVE, calling with its mouth. Keep in step with wave and call in mascot.css.
const WAVE = 6000;
// While working the mascot blinks once every BLINK. Keep in step with blink in mascot.css.
const BLINK = 4000;
// When the user interrupts it the mascot sulks: nose in the air, it turns its
// face to one side and then the other, SWING ms per side. Keep in step with
// sulk-swing and sulk-puff-* in mascot.css.
const SWING = 4000;
// A spell while working (see SPELLS): it waves the wand about
// with tiny zigzag bolts flying, raises it straight up and zaps itself,
// flashes its skeleton three times, ends up sooty, shakes the soot off and is
// clean again, all in ZAP. Keep in step with zap-* in mascot.css.
const ZAP = 8400;
// Reading before every spell (see life.js): it takes out its spellbook, opens it,
// reads two lines (its eyes running along them), turns the page, reads two
// more, closes the book and puts it away, all in READ. Keep in step with
// read-* in mascot.css.
const READ = 6000;
// A spell while working: it waves the wand about and a breeze swirls around
// it, then a gust blows its hat off. The hat tumbles through the air and
// lands on the ground to the right; it stares at it, runs over, picks it up,
// puts it back on and runs back to where it started, all in GUST. Its moves
// are in GUST_MOVES below; keep in step with gust-* in mascot.css.
const GUST = 8000;
// A spell while working: it waves the wand
// about with fiery sparks flying and shoots a fireball off to the top right,
// past the edge of the screen. A spark from the fireball drops onto its hat,
// which burns for a good while before it notices; then it gets a fright, jumps
// and puts the fire out with water from its wand, all in FIRE. Keep in step
// with fire-* in mascot.css.
const FIRE = 10000;
// A spell while working: it waves the wand
// about with droplets flying, and a stream of water shoots off to the top
// right. All is quiet for a couple of seconds, then the water comes down on
// it as rain; it gets soaked and cross, dries off and is fine again, all in
// RAIN. Keep in step with rain-* in mascot.css.
const RAIN = 12000;
// The rain spell as a prank (see life.js): with friends of its group asleep,
// the rain comes down on some of them instead, PRANK_AT into RAIN, and
// LAUGH_AFTER later it laughs at them. A friend it lands on is soaked for SOAK: woken with a
// start, wet and cross, glaring at the prankster, it dries off and goes back
// to sleep. Keep in step with prank-* and soak-* in mascot.css.
const PRANK_AT = 6600;
const LAUGH_AFTER = 1500;
const SOAK = 5000;
// The gust spell as a prank (see life.js): the wind ("blow", the gust without
// losing its own hat) reaches its friends asleep SWAP_AT into GUST and swaps
// the hats of two of them, or of one of them and its own ("blowswap"). For
// SWAP from then a "hatswap" mascot: the wind leans it over and takes its
// hat; the hats fly over (SWAP_FLY, see flyHats) and land on the wrong heads,
// crooked (SWAP_TILT degrees); it wakes, looks up at the hat, then at the
// other one; they toss the hats back (SWAP_BACK), and it falls asleep again.
// Keep in step with swap-* in mascot.css.
const SWAP_AT = 2400;
const SWAP = 8400;
const SWAP_FLY = [200, 2200];
const SWAP_BACK = [5000, 5800];
const SWAP_TILT = 14;
// The fire spell as a prank (see life.js): with friends asleep by the
// group's campfire, the prankster ("fireprank", FIRE_PRANK long) shoots the
// fireball straight up off the screen, and a good while later it comes back
// down in the fire, landing BLAZE_AT in (see dropFireball). The fire flares
// up, the friends by it are "startled" for STARTLE, and the prankster looks
// ashamed. Keep in step with fp-* and startle-* in mascot.css.
const FIRE_PRANK = 12000;
const BLAZE_AT = 7200;
const STARTLE = 4000;
// The spells a working mascot casts after reading its spellbook (see
// life.js), one picked at random each time: the status mascotSVG draws it
// with, and how long it plays. A new spell: draw it in mascotSVG, animate it
// in mascot.css and add it here.
const SPELLS = { zap: ZAP, gust: GUST, fire: FIRE, rain: RAIN };
// Going into the sulk from work, and out of it back to work, slides the face
// over this long.
const SULK_MS = 500;

// Changes between sleeping and awake play a short animation of eye frames
// instead of jumping straight to the new state. Each frame is [eyes, ms]:
// "open", "lid1" to "lid3" (eyelids part way down), "line" (lids shut over
// the open eyes) or "closed" (the sleeping face). The CSS keyframes are made
// from this list below, so the timing lives only here.
const TRANSITIONS = {
  // asleep, the lids lift bit by bit, two blinks, wide awake
  wake: { frames: [["closed", 400], ["lid3", 120], ["lid2", 120], ["lid1", 100], ["open", 220],
                   ["line", 110], ["open", 240], ["line", 100], ["open", 390]] },
  // awake for a moment, the lids droop slowly, lift once as it fights sleep,
  // droop again and shut
  sleep: { frames: [["open", 400], ["lid1", 180], ["lid2", 200], ["lid3", 260], ["lid2", 160],
                    ["lid1", 180], ["lid2", 220], ["lid3", 280], ["line", 220], ["closed", 300]] },
};
for (const tr of Object.values(TRANSITIONS)) tr.ms = tr.frames.reduce((sum, [, ms]) => sum + ms, 0);
// How long a change between states plays: one of the eye-frame transitions
// above, or sliding into ("sulk") or out of ("unsulk") a sulk.
const changeMs = kind => TRANSITIONS[kind] ? TRANSITIONS[kind].ms : SULK_MS;

// How far down the eyelid is for each eye frame, as a share of the eye's height.
const LIDS = { lid1: .3, lid2: .5, lid3: .7 };

// One keyframes rule per transition frame: hidden, shown during its slot,
// hidden again (the last frame stays). Played with step-end, so frames switch
// sharply like a flipbook.
(function addTransitionKeyframes() {
  let css = "";
  for (const [kind, tr] of Object.entries(TRANSITIONS)) {
    let at = 0;
    tr.frames.forEach(([, ms], i) => {
      const from = (at / tr.ms) * 100, to = ((at + ms) / tr.ms) * 100;
      const last = i === tr.frames.length - 1;
      css += `@keyframes tr-${kind}-${i} { 0% { opacity: ${from === 0 ? 1 : 0} } ${from}% { opacity: 1 } ` +
        (last ? "100% { opacity: 1 } }\n" : `${to}% { opacity: 0 } 100% { opacity: 0 } }\n`);
      at += ms;
    });
  }
  const style = document.createElement("style");
  style.textContent = css;
  document.head.appendChild(style);
})();

// The gust spell's moves (see GUST), as [ms, x, y] on the canvas: where the
// body (with the wand) is, and where the hat is, with its turn in degrees.
// Once the hat is back on its head it follows the body point by point, so the
// two keyframes rules are made from these lists below and never part.
const GUST_MOVES = (() => {
  // Running from x0 to x1 between t0 and t1 in n quick hops, 3 units high.
  const hops = (t0, t1, x0, x1, n) => Array.from({ length: 2 * n }, (_, i) => {
    const k = (i + 1) / (2 * n);
    return [t0 + k * (t1 - t0), x0 + k * (x1 - x0), i % 2 ? 0 : -3];
  });
  // Staring at the hat, it runs over (to 5.1 s), crouches and picks it up,
  // gives a little bob as it's put on (5.75 s), and runs back (6-7.2 s).
  const body = [[0, 0, 0], [3900, 0, 0], ...hops(3900, 5100, 0, 54, 6),
    [5250, 54, 2], [5450, 54, 0], [5750, 54, 1.5], [5900, 54, 0], [6000, 54, 0],
    ...hops(6000, 7200, 54, 0, 6), [GUST, 0, 0]];
  // Lifted off at 2.45 s, it tumbles up and over to the right, lands on its
  // brim on the ground (3.5 s; "ground" goes down less for a hat pulled down,
  // see hatDy) and rocks once. Picked up at 5.25 s, it's on the head at 5.7 s.
  const ground = dy => `calc(${29 + dy}px - var(--hat-dy, 0px))`;
  const hat = [[0, 0, 0, 0], [2450, 0, 0, 0], [2650, 10, -14, -25], [2900, 29, -23, 40],
    [3150, 50, -16, 110], [3350, 67, 4, 220], [3500, 76, ground(0), 360],
    [3620, 76, ground(-3), 352], [3760, 76, ground(0), 360], [5250, 76, ground(0), 360],
    [5450, 64, -10, 370], [5700, 54, 0, 360],
    ...body.filter(([ms]) => ms > 5700).map(p => [...p, 360])];
  return { body, hat };
})();

(function addGustKeyframes() {
  const px = v => typeof v === "number" ? `${+v.toFixed(2)}px` : v;
  const rule = (name, points) => `@keyframes ${name} {` + points.map(([ms, x, y, turn]) =>
    ` ${+(ms / GUST * 100).toFixed(3)}% { transform: translate(${px(x)}, ${px(y)})${turn === undefined ? "" : ` rotate(${turn}deg)`} }`).join("") + " }\n";
  const style = document.createElement("style");
  style.textContent = rule("gust-run", GUST_MOVES.body) + rule("gust-hat", GUST_MOVES.hat);
  document.head.appendChild(style);
})();

// The rain spell's stream of water (see RAIN): a curve from the wand's tip
// (raised a little) off past the top right of the screen, as the four points
// of a cubic Bezier. bezier gives a point on it at t (k 0 for x, 1 for y),
// bezierSlope its direction there. As a prank (see PRANK_AT) the stream goes
// straight up instead (RAIN_UP), from the wand raised upright.
const RAIN_ARC = [[63.5, 30], [64, 10], [80, -12], [112, -28]];
const RAIN_UP = [[55, 28], [55.6, 4], [53.8, -30], [55, -72]];
const bezier = (P, t, k) => (1 - t) ** 3 * P[0][k] + 3 * (1 - t) ** 2 * t * P[1][k] + 3 * (1 - t) * t * t * P[2][k] + t ** 3 * P[3][k];
const bezierSlope = (P, t, k) => 3 * (1 - t) ** 2 * (P[1][k] - P[0][k]) + 6 * (1 - t) * t * (P[2][k] - P[1][k]) + 3 * t * t * (P[3][k] - P[2][k]);

// The splash at the head of the stream rides along the curve as the stream
// grows (2.8-3.6 s of RAIN), turned the way it flows. The stream is revealed
// evenly along its length, so the head's keyframes are spaced by length too.
(function addRainKeyframes() {
  // One rule per stream: rain-head along RAIN_ARC, prank-head along RAIN_UP.
  const headRule = (name, P) => {
    const N = 200, pts = [], lens = [0];
    for (let i = 0; i <= N; i++) pts.push([bezier(P, i / N, 0), bezier(P, i / N, 1)]);
    for (let i = 1; i <= N; i++) lens.push(lens[i - 1] + Math.hypot(pts[i][0] - pts[i - 1][0], pts[i][1] - pts[i - 1][1]));
    const pose = share => {
      const i = Math.max(0, lens.findIndex(l => l >= share * lens[N]));
      const t = i / N, turn = Math.atan2(bezierSlope(P, t, 1), bezierSlope(P, t, 0)) * 180 / Math.PI;
      return `transform: translate(${pts[i][0].toFixed(2)}px, ${pts[i][1].toFixed(2)}px) rotate(${turn.toFixed(1)}deg)`;
    };
    const [from, to] = [2800 / RAIN * 100, 3600 / RAIN * 100];
    let rule = `@keyframes ${name} { 0%, ${(from - .01).toFixed(2)}% { opacity: 0; ${pose(0)} }`;
    for (let j = 0; j <= 10; j++) rule += ` ${(from + (to - from) * j / 10).toFixed(2)}% { opacity: 1; ${pose(j / 10)} }`;
    return rule + ` ${(to + .01).toFixed(2)}%, 100% { opacity: 0; ${pose(1)} } }\n`;
  };
  let css = headRule("rain-head", RAIN_ARC) + headRule("prank-head", RAIN_UP);
  // The prankster shakes with laughter for 3 s from LAUGH_AFTER into the
  // rain, a bounce every 150 ms.
  css += "@keyframes prank-shake { 0% { transform: translateY(0) }";
  for (let ms = PRANK_AT + LAUGH_AFTER, i = 0; ms <= PRANK_AT + LAUGH_AFTER + 3000; ms += 150, i++) {
    css += ` ${(ms / RAIN * 100).toFixed(2)}% { transform: translateY(${i % 2 ? -1.4 : 0}px) }`;
  }
  css += " 100% { transform: translateY(0) } }\n";
  const style = document.createElement("style");
  style.textContent = css;
  document.head.appendChild(style);
})();

// A four-point sparkle centred on x, y.
const sparkle = (x, y, r, fill) =>
  `<path d="M${x} ${y-r} Q${x} ${y} ${x+r} ${y} Q${x} ${y} ${x} ${y+r} Q${x} ${y} ${x-r} ${y} Q${x} ${y} ${x} ${y-r}Z" fill="${fill}"/>`;
// A wizard hat in color on the mascot's canvas, pulled down by dy (a shape's hatDy).
const hatSVG = (color, dy = 0) => `<g transform="translate(0 ${dy})">` +
  `<path d="M14 26 C21 21 25 11 36 2 C34 11 38 19 46 26Z" fill="${color}"/>` +
  `<ellipse cx="30" cy="27" rx="22" ry="4" fill="${color}"/>` +
  `<ellipse cx="30" cy="26" rx="22" ry="3" fill="#000" opacity=".15"/>` +
  sparkle(31, 17, 3.5, MAGIC.Y) + `</g>`;
// The same shapes again, mirrored to the right side of the body.
const mirrored = svg => svg + `<g transform="matrix(-1 0 0 1 60 0)">${svg}</g>`;
// A darker coat over a shape, for the snail's shell.
const darker = (svg, o) => svg + svg.replace(/fill="[^"]*"/g, `fill="#000" opacity="${o}"`);
// A spiral from cx, cy outwards, for the snail's shell.
const spiral = (cx, cy, turns, step) => {
  let d = "";
  for (let a = 0; a <= turns * 2 * Math.PI; a += .3) {
    const r = 1 + step * a;
    d += `${d ? "L" : "M"}${(cx + r * Math.cos(a)).toFixed(1)} ${(cy + r * Math.sin(a)).toFixed(1)} `;
  }
  return d;
};

// Four creatures, taken in turn by mascot index. The canvas is 78x60: the body
// is centred on x 30 under the hat, the wand sticks out to the right. Each has
// a body path, where the eyes sit (y, half the gap between them), and
// optionally feet, extras drawn behind the body, the hat moved down (hatDy)
// and cheeks closer to the middle (cheekX), or a sulking face that turns less
// far to each side and up (sulk: [x, y]) so it stays on the face.
const SHAPES = [
  { // wisp: a floating spirit with a tail curling off to the left
    eyeY: 40, gap: 8,
    path: "M10 42 C10 30 18 26 30 26 C42 26 50 30 50 42 C50 52 42 56 32 56 C24 56 18 57 11 60 C14 56 14 54 12 51 C10 48 10 45 10 42Z" },
  { // cat: ears poking out under the hat brim, tail on the left
    eyeY: 41, gap: 9, feet: true,
    path: "M6 46 C6 32 16 26 30 26 C44 26 54 32 54 46 C54 53 49 57 42 57 L18 57 C11 57 6 53 6 46Z",
    behind: c => mirrored(`<path d="M7 37 L1 21 L16 30Z" fill="${c}" stroke="${c}" stroke-width="2" stroke-linejoin="round"/>`) +
      `<path d="M8 52 Q-1 51 2 40" fill="none" stroke="${c}" stroke-width="3.5" stroke-linecap="round"/>` },
  { // frog: wide and low, the eyes on bumps either side of the hat
    eyeY: 38, gap: 23, feet: true, hatDy: 6, cheekX: 13, sulk: [2, 0],
    path: "M2 50 C2 40 12 34 30 34 C48 34 58 40 58 50 C58 55 52 57 46 57 L14 57 C8 57 2 55 2 50Z",
    behind: c => mirrored(`<circle cx="7" cy="38" r="6.5" fill="${c}"/>`) },
  { // snail: a shell on its back and a long foot
    eyeY: 41, gap: 5, cheekX: 6, sulk: [1.5, 4],
    path: "M21 44 C21 32 24 28 30 28 C36 28 39 32 39 44 L39 52 C46 52 49 57 44 57 L5 57 C2 57 2 53 6 52 L21 52Z",
    behind: c => darker(`<circle cx="12" cy="43" r="10.5" fill="${c}"/>`, .2) +
      `<path d="${spiral(12, 43, 2.3, 1.05)}" fill="none" stroke="#000" stroke-opacity=".3" stroke-width="1.6" stroke-linecap="round"/>` },
];

// A soft shadow on the lower part of the body.
const SHADE = `<defs><linearGradient id="mascot-shade" x1="0" y1="0" x2="0" y2="1">` +
  `<stop offset=".7" stop-color="#000" stop-opacity="0"/><stop offset="1" stop-color="#000" stop-opacity=".22"/></linearGradient></defs>`;

// mascotSVG draws the mascot for an agent. change, when set, says the agent
// went from asleep to awake (kind "wake") or back ("sleep"), into a sulk
// ("sulk") or out of it ("unsulk") ago ms ago; while that change lasts the
// mascot plays it instead of its status animation. change.fromY /
// change.fromOpacity, when known, are where the mascot was in its hop at that
// moment, so it eases back from there, and change.fromFaceX which side its
// face was turned to in a sulk.
// poseOf reads where a drawn mascot is in its hop (and sulk) right now, so a
// change that starts at this moment can ease back from there.
function poseOf(svg) {
  if (!svg) return {};
  const st = getComputedStyle(svg);
  const m = /matrix\(([^)]+)\)/.exec(st.transform);
  const face = svg.querySelector(".sulk-swing");
  return { fromY: m ? Number(m[1].split(",")[5]) : 0, fromOpacity: Number(st.opacity),
           fromFaceX: face ? new DOMMatrix(getComputedStyle(face).transform).e : undefined };
}

// opts.side, for a mascot "soaked" by a prank, is where the prankster is:
// -1 to its left, 1 to its right; for one in a hat swap ("hatswap",
// "blowswap") where the other one of the swap is, and opts.other is that
// one's mascot index (whose hat it gets); for one in a fire prank
// ("fireprank", "startled") where the campfire is.
function mascotSVG(index, status, change = null, opts = {}) {
  const color = COLORS[index % COLORS.length];
  const shape = SHAPES[index % SHAPES.length];
  // Animations are timed by a clock (a negative delay), so redrawing the card
  // doesn't restart them. After a change the clock starts when the change
  // ends, so the hop, breathing and Zzz begin at the start of their cycle,
  // right where the change left the mascot; otherwise it's the wall clock.
  const since = change ? Math.max(0, change.ago - changeMs(change.kind)) : Date.now();
  const clock = period => -(since % period);
  const asleep = status === "done" || status === "idle";
  // The change playing right now, if any; moving is set for the eye-frame ones.
  const changing = change && change.ago < changeMs(change.kind) ? change.kind : null;
  const moving = changing && TRANSITIONS[changing];
  const sulky = status === "interrupted" || changing === "sulk";
  const hat = HATS[index % HATS.length];

  // Eyes for one frame (see TRANSITIONS): open ones have a small shine, a lid
  // part way down covers them with the body color, closed ones are a sleepy arc.
  const drawEyes = eyes => [30 - shape.gap, 30 + shape.gap].map(x => {
    const y = shape.eyeY, rx = 3, ry = 3.6;
    if (eyes === "closed") return `<path d="M${x-3.5} ${y} Q${x} ${y+3} ${x+3.5} ${y}" fill="none" stroke="${EYE}" stroke-width="1.8" stroke-linecap="round"/>`;
    if (eyes === "line") return `<path d="M${x-3.2} ${y+.5} L${x+3.2} ${y+.5}" stroke="${EYE}" stroke-width="1.8" stroke-linecap="round"/>`;
    let eye = `<ellipse cx="${x}" cy="${y}" rx="${rx}" ry="${ry}" fill="${EYE}"/><circle cx="${x+1}" cy="${y-1.4}" r="1" fill="#fff"/>`;
    if (LIDS[eyes]) {
      const edge = y - ry + 2 * ry * LIDS[eyes];
      const half = rx * Math.sqrt(1 - ((edge - y) / ry) ** 2);
      // The lid is the part of the eye above its edge, so it never sticks out past the eye.
      const big = edge > y ? 1 : 0;
      eye += `<path d="M${(x-half).toFixed(2)} ${edge.toFixed(2)} A${rx} ${ry} 0 ${big} 1 ${(x+half).toFixed(2)} ${edge.toFixed(2)}Z" fill="${color}" stroke="${color}" stroke-width=".8"/>` +
        `<path d="M${(x-half).toFixed(2)} ${edge.toFixed(2)} L${(x+half).toFixed(2)} ${edge.toFixed(2)}" stroke="${EYE}" stroke-width="1.2" stroke-linecap="round"/>`;
    }
    return eye;
  }).join("");

  const drawHat = () => hatSVG(hat, shape.hatDy);

  // drawBody draws the body with its extras, face and hat. eyesWrap, when
  // set, wraps the eyes (e.g. in a blinking group); withHat false leaves the
  // hat off (it's drawn on its own).
  const drawBody = (eyes, eyesWrap = e => e, withHat = true) => {
    const cheekX = shape.cheekX ?? shape.gap + 7;
    return (shape.feet ? [18, 36].map(x => `<rect x="${x}" y="52" width="6" height="7" rx="3" fill="${color}"/><rect x="${x}" y="52" width="6" height="7" rx="3" fill="#000" opacity=".18"/>`).join("") : "") +
      (shape.behind ? shape.behind(color) : "") +
      `<path d="${shape.path}" fill="${color}"/><path d="${shape.path}" fill="url(#mascot-shade)"/>` +
      [30 - cheekX, 30 + cheekX].map(x => `<ellipse cx="${x}" cy="${shape.eyeY+6}" rx="3.2" ry="2" fill="#fff" opacity=".18"/>`).join("") +
      eyesWrap(drawEyes(eyes)) + (withHat ? drawHat() : "");
  };

  // The sulking face: narrowed eyes and a pout. A "hmph" puff pops out on
  // the side the face turns to (side -1 is left, 1 right).
  const y = shape.eyeY;
  const sulkFace = drawEyes("lid2") +
    `<path d="M28 ${y + 6} Q30 ${y + 4.5} 32 ${y + 6}" fill="none" stroke="${EYE}" stroke-width="1.5" stroke-linecap="round"/>`;
  const puff = side => [[7, 35, 2.2], [4.5, 30.5, 1.6], [3, 27, 1.1]].map(([cx, cy, r]) =>
    `<circle cx="${side < 0 ? cx : 60 - cx}" cy="${cy}" r="${r}" fill="#b4b2a9"/>`).join("");

  // How far the sulking face turns to each side and up, for mascot.css.
  const [sx, sy] = shape.sulk || [4, 4];
  const sulkTo = `--sx:${sx}px;--sy:${-sy}px`;

  // The body and its extras in one color, laid over the body.
  const coat = c => (shape.feet ? [18, 36].map(x => `<rect x="${x}" y="52" width="6" height="7" rx="3" fill="${c}"/>`).join("") : "") +
    (shape.behind ? shape.behind(c) : "") + `<path d="${shape.path}" fill="${c}"/>`;

  // Rained on (the rain spell, or soaked by a prank; pre is the class prefix
  // that times it): a wet coat darkens the body, the eyes glare under cross
  // brows with a frown, and the soggy hat droops. rainOn draws the rain
  // falling from above, drips running off and steam as it dries.
  const crossBrows = [-1, 1].map(side => {
    const x = 30 + side * shape.gap;
    return `<path d="M${x + side * 3.5} ${y - 6.8} L${x - side * 2.8} ${y - 4.6}" stroke="${EYE}" stroke-width="1.6" stroke-linecap="round"/>`;
  }).join("") + `<path d="M27 ${y + 7.2} Q30 ${y + 5.4} 33 ${y + 7.2}" fill="none" stroke="${EYE}" stroke-width="1.5" stroke-linecap="round"/>`;
  const wetBody = (pre, eyes = e => e) => drawBody("open", e => `<g class="${pre}-wet">${coat("#1f4f7a")}</g>` +
    `${eyes(`<g class="${pre}-eyes">${e}</g>`)}<g class="${pre}-cross">${crossBrows}</g>`, false) +
    `<g class="${pre}-hat" style="transform-origin:30px ${27 + (shape.hatDy || 0)}px">${drawHat()}</g>`;
  const rainOn = pre =>
    [[15, 0], [19, 220], [23, 90], [27, 300], [31, 160], [35, 40], [39, 260], [43, 340], [21, 130], [33, 200], [29, 60], [41, 120]].map(([x, d]) =>
      `<path class="${pre}-drop" style="animation-delay:${d}ms" d="M${x} -8 v4" stroke="#9fd8f5" stroke-width="1.2" stroke-linecap="round"/>`).join("") +
    [[11, 50, 0], [49, 48, 500], [24, 57, 250], [37, 57, 750]].map(([x, y, d]) =>
      `<circle class="${pre}-drip" style="animation-delay:${d}ms" cx="${x}" cy="${y}" r=".9" fill="#9fd8f5"/>`).join("") +
    [-1, 0, 1].map(ux => `<circle class="${pre}-steam" style="--dx:${ux * 4}px;animation-delay:${(ux + 1) * 200}ms" cx="${30 + ux * 10}" cy="30" r="2.2" fill="#e4e1da"/>`).join("");

  // zapLayers draws what the zap gag shows over the body: the x-ray (the
  // body and its extras in dark blue with a skull, spine and ribs) and the
  // soot (the body in soot with only the whites of the eyes left).
  const zapLayers = () => {
    const BONE = "#eef1f7", XRAY = "#1d2240", SOOT = "#4a423c";
    const eyes = [30 - shape.gap, 30 + shape.gap];
    // The skull sits behind the eyes, no wider than a head (the frog's eyes
    // stick out on bumps) and below a hat that's pulled down.
    const socket = Math.min(shape.gap, 9), sy = y + (shape.hatDy || 0) * .7;
    const ribW = Math.min(12, shape.gap + 3);
    let bones = `<line x1="30" y1="${sy + 6}" x2="30" y2="54" stroke="${BONE}" stroke-width="2.2" stroke-linecap="round"/>`;
    for (let r = sy + 10; r < 55; r += 3.6) {
      bones += `<path d="M${30 - ribW} ${r + 1.5} Q30 ${r - 2} ${30 + ribW} ${r + 1.5}" fill="none" stroke="${BONE}" stroke-width="1.6" stroke-linecap="round"/>`;
    }
    bones += `<ellipse cx="30" cy="${sy}" rx="${socket + 4}" ry="7" fill="${BONE}"/>` +
      [30 - socket, 30 + socket].map(x => `<ellipse cx="${x}" cy="${sy}" rx="2.6" ry="3" fill="${XRAY}"/>`).join("") +
      `<path d="M29 ${sy + 3.4} L31 ${sy + 3.4} L30 ${sy + 1.8}Z" fill="${XRAY}"/>` +
      `<path d="M26.5 ${sy + 5.6} H33.5 M28 ${sy + 4.7} V${sy + 6.5} M30 ${sy + 4.7} V${sy + 6.5} M32 ${sy + 4.7} V${sy + 6.5}" stroke="${XRAY}" stroke-width=".8"/>`;
    const sootEyes = eyes.map(x => `<circle cx="${x}" cy="${y}" r="3.2" fill="#fff"/><circle cx="${x}" cy="${y + .6}" r="1.4" fill="${EYE}"/>`).join("");
    return `<g class="zap-skel">${coat(XRAY)}${bones}</g><g class="zap-soot">${coat(SOOT)}${sootEyes}</g>`;
  };

  let out = SHADE;
  if (changing === "sulk" || changing === "unsulk") {
    // Into the sulk the face slides up to the left and swaps to the sulking
    // face; out of it, it slides back from whichever side it was turned to
    // and opens its eyes.
    const t = `animation-duration:${SULK_MS}ms;animation-delay:${-change.ago}ms`;
    const into = changing === "sulk";
    out += drawBody("open", open =>
      `<g class="${changing}" style="${t};${sulkTo};--from-fx:${change.fromFaceX ?? -sx}px">` +
      `<g class="${into ? "swap-out" : "swap-in"}" style="${t}">${open}</g>` +
      `<g class="${into ? "swap-in" : "swap-out"}" style="${t}">${sulkFace}</g></g>`);
  } else if (moving) {
    out += moving.frames.map(([eyes], i) =>
      `<g class="tr" style="animation-name:tr-${change.kind}-${i};animation-duration:${moving.ms}ms;animation-delay:${-change.ago}ms">${drawBody(eyes)}</g>`).join("");
  } else if (asleep) {
    // A sleeping mascot snores: the body and hat swell and sink with each breath.
    out += `<g class="snore" style="animation-delay:${clock(SNORE)}ms">${drawBody("closed")}</g>`;
  } else {
    // A working mascot blinks now and then: the eyes squash shut for a moment.
    if (status === "working") {
      out += drawBody("open", e => `<g class="blink" style="animation-delay:${clock(BLINK)}ms">${e}</g>`);
    } else if (status === "looking") {
      // Between jobs (see life.js): it looks around for company, eyes to one
      // side and then the other. Plays once, from when it's drawn.
      out += drawBody("open", e => `<g class="look">${e}</g>`);
    } else if (status === "read") {
      // Reading (see READ): the eyes look down at the book and run along the lines.
      out += drawBody("open", e => `<g class="read-eyes">${e}</g>`);
    } else if (status === "zap") {
      // Zapped (see ZAP): between the eyes and the hat lie an x-ray of it
      // (dark body, white bones) and a coat of soot, each shown in its turn.
      out += `<g class="zap-shake"><g class="zap-wiggle">${drawBody("open", e => e + zapLayers())}</g></g>`;
    } else if (status === "gust") {
      // Hat blown off (see GUST): the body runs (gust-run) and is pushed by
      // the gust (gust-lean); the eyes follow the wand, then the hat. The hat
      // is drawn on its own, with the wand.
      out += `<g class="gust-run"><g class="gust-lean">${drawBody("open", e => `<g class="gust-eyes">${e}</g>`, false)}</g></g>`;
    } else if (status === "hatswap" || status === "blowswap") {
      // A hat swap (see SWAP): its own hat, then the other one's crooked,
      // then its own again (swap-own, swap-other). Asleep in it ("hatswap"),
      // it leans in the wind, wakes, looks up at the hat and then at the other
      // one, hops as it tosses the hat back, and falls asleep again. Casting
      // it ("blowswap"), awake, it only looks; its swap starts SWAP_AT in.
      const swap = status === "hatswap", side = opts.side || 1, hd = shape.hatDy || 0;
      const late = swap ? "" : `animation-delay:${SWAP_AT}ms`;
      const hats = `<g class="swap-own" style="${late}">${drawHat()}</g>` +
        `<g class="swap-other" style="${late}"><g style="transform-box:view-box;transform-origin:30px ${16 + hd}px;transform:translateX(${-side * 2}px) rotate(${-side * SWAP_TILT}deg)">` +
        `${hatSVG(HATS[(opts.other || 0) % HATS.length], hd)}</g></g>`;
      out += swap
        ? `<g class="swap-lean" style="--side:${side}"><g class="swap-hop">` +
          drawBody("open", e => `<g class="swap-shut">${drawEyes("closed")}</g><g class="swap-eyes" style="--side:${side}">${e}</g>`, false) +
          hats + `</g></g>`
        : drawBody("open", e => `<g class="swapc-eyes" style="--side:${side};${late}">${e}</g>`, false) + hats;
    } else if (status === "blow") {
      // Casting the gust at its friends (see SWAP_AT): it keeps its hat.
      out += drawBody("open");
    } else if (status === "fire") {
      // Hat on fire (see FIRE): the eyes follow the fireball and the spark,
      // then go wide with fright and the mouth gasps; it jumps (fire-jump).
      out += `<g class="fire-jump">${drawBody("open", e => `<g class="fire-eyes">${e}</g>` +
        `<ellipse class="fire-gasp" cx="30" cy="${y + 7}" rx="2" ry="2.4" fill="${EYE}"/>`)}</g>`;
    } else if (status === "fireprank") {
      // The fire spell as a prank (see BLAZE_AT): the eyes follow the wand
      // and the fireball up, wait looking up and about, follow the ball down
      // into the campfire as it flares up; then it looks ashamed: eyes down
      // and away, cheeks red, a wobbly mouth, a bead of sweat.
      const side = opts.side || 1;
      const shame = `<g transform="translate(${-side * 1.4} 1.2)">${drawEyes("lid1")}</g>` +
        [30 - (shape.cheekX ?? shape.gap + 7), 30 + (shape.cheekX ?? shape.gap + 7)].map(x =>
          `<ellipse cx="${x}" cy="${y + 5.5}" rx="3.6" ry="2.1" fill="#ff7aa2" opacity=".6"/>`).join("") +
        `<path d="M27 ${y + 7.4} q1.5 -1.2 3 0 q1.5 1.2 3 0" fill="none" stroke="${EYE}" stroke-width="1.3" stroke-linecap="round"/>` +
        `<path d="M${30 - side * 16} ${y - 9} q-1.4 2.4 0 3.2 q1.4 -.8 0 -3.2Z" fill="#9fd8f5"/>`;
      out += drawBody("open", e => `<g class="fp-eyes" style="--side:${side}">${e}</g><g class="fp-shame">${shame}</g>`);
    } else if (status === "startled") {
      // Startled by a fire prank (see STARTLE), asleep by the campfire: the
      // eyes snap open wide at the fire (--side), the mouth gasps, it jumps
      // and trembles, calms down and falls asleep again.
      out += `<g class="startle-jump">${drawBody("open", e => `<g class="startle-shut">${drawEyes("closed")}</g>` +
        `<g class="startle-eyes" style="--side:${opts.side || 1}">${e}</g>` +
        `<ellipse class="startle-gasp" cx="30" cy="${y + 7}" rx="2" ry="2.4" fill="${EYE}"/>`)}</g>`;
    } else if (status === "rain") {
      // Rained on by its own spell (see RAIN).
      out += wetBody("rain");
    } else if (status === "soaked") {
      // Soaked by a prank (see SOAK): asleep at first, it wakes with a start,
      // glares towards the prankster (--side), and in the end its eyes close
      // again.
      out += wetBody("soak", e => `<g class="soak-shut">${drawEyes("closed")}</g>` +
        `<g style="--side:${opts.side || 0}">${e}</g>`);
    } else if (status === "prank") {
      // The rain spell as a prank (see PRANK_AT): while it rains on its
      // friends it shakes with laughter, eyes squeezed shut, mouth wide open,
      // tears of laughter.
      const laugh = [30 - shape.gap, 30 + shape.gap].map((x, i) =>
        `<path d="M${x - 3.5} ${y + 1.2} Q${x} ${y - 3.4} ${x + 3.5} ${y + 1.2}" fill="none" stroke="${EYE}" stroke-width="1.8" stroke-linecap="round"/>` +
        `<path d="M${x + (i ? 4 : -4)} ${y + 1} q-1 2 0 2.6 q1 -.6 0 -2.6Z" fill="#9fd8f5"/>`).join("") +
        `<path d="M26.3 ${y + 4.6} Q30 ${y + 12} 33.7 ${y + 4.6}Z" fill="${EYE}"/>` +
        `<path d="M28 ${y + 8.4} Q30 ${y + 6.8} 32 ${y + 8.4} Q30 ${y + 9.6} 28 ${y + 8.4}Z" fill="#d4537e"/>`;
      out += `<g class="prank-shake">${drawBody("open", e => `<g class="prank-eyes">${e}</g><g class="prank-laugh">${laugh}</g>`)}</g>`;
    } else if (status === "interrupted") {
      // Sulking: nose in the air, the face turns from one side to the other
      // every SWING, with a "hmph" puff on each turn. The wand is put away.
      const d = `animation-delay:${clock(2 * SWING)}ms`;
      out += drawBody("open", () => `<g class="sulk-swing" style="${d};${sulkTo}">${sulkFace}</g>` +
        `<g class="sulk-puff-l" style="${d}">${puff(-1)}</g><g class="sulk-puff-r" style="${d}">${puff(1)}</g>`);
    } else {
      out += drawBody("open");
    }
  }
  const wand = `<line x1="55" y1="48" x2="66" y2="34" stroke="${MAGIC.B}" stroke-width="3" stroke-linecap="round"/>` + sparkle(67, 32, 3.5, MAGIC.Y);
  if (status === "waiting" && !changing) {
    // A sign on a stick instead of the wand, waved from where the stick is held.
    out += `<g class="sign" style="animation-delay:${clock(WAVE)}ms">` +
      `<line x1="56" y1="50" x2="64" y2="22" stroke="${MAGIC.B}" stroke-width="3" stroke-linecap="round"/>` +
      `<rect x="54" y="4" width="22" height="19" rx="3" fill="#f1efe8" stroke="#000" stroke-opacity=".25" stroke-width="1.2"/>` +
      `<text x="65" y="19.5" text-anchor="middle" font-size="16" font-weight="800" fill="${EYE}" font-family="system-ui, sans-serif">!</text></g>` +
      // With each wave the mouth opens twice, as if it calls "hey, hey!".
      `<ellipse class="call" cx="30" cy="${shape.eyeY + 7}" rx="2.6" ry="2.4" fill="${EYE}" style="animation-delay:${clock(WAVE)}ms"/>`;
  } else if (status === "read") {
    // The wand is put away while it reads, and the book is held in front of
    // it, just below the eyes: closed (a cover with a star) or open (two
    // pages of lines). Turning the page, a sheet lifts off the right page
    // and lands on the left one: its front going up, its back coming down.
    const by = shape.eyeY + 6, mid = by + 6;
    const lines = x => [3.5, 6, 8.5].map(dy => `<path d="M${x} ${by + dy} h7.5" stroke="#b5a98f" stroke-width=".8" stroke-linecap="round"/>`).join("");
    const page = (side, fill = "#f3ead7", edge = .12) => `<path d="M30 ${by + 1} Q${30 + side * 6} ${by - .8} ${30 + side * 12} ${by + .4} V${by + 11} Q${30 + side * 6} ${by + 9.6} 30 ${by + 11.6}Z" fill="${fill}" stroke="#000" stroke-opacity="${edge}" stroke-width=".5"/>` +
      lines(side < 0 ? 20.5 : 32);
    out += `<g class="read-wand">${wand}</g>` +
      `<g class="read-book" style="transform-origin:30px ${mid}px">` +
      `<g class="read-closed" style="transform-origin:30px ${mid}px"><rect x="22" y="${by}" width="16" height="12.5" rx="1.6" fill="#8a2f3a"/>` +
      `<rect x="22" y="${by}" width="2.4" height="12.5" rx="1" fill="#000" opacity=".25"/>${sparkle(31.2, mid + .2, 3, MAGIC.Y)}</g>` +
      `<g class="read-open" style="transform-origin:30px ${mid}px"><rect x="16.8" y="${by + .6}" width="26.4" height="12" rx="1.6" fill="#8a2f3a"/>` +
      `${page(-1)}${page(1)}<line x1="30" y1="${by + 1}" x2="30" y2="${by + 11.6}" stroke="#000" stroke-opacity=".15" stroke-width=".6"/>` +
      `<g class="read-flip-up" style="transform-origin:30px ${mid}px">${page(1, "#f6efe0", .3)}</g>` +
      `<g class="read-flip-down" style="transform-origin:30px ${mid}px">${page(-1, "#e8dcc2", .3)}</g></g></g>`;
  } else if (status === "zap") {
    // The wand is waved about with tiny bolts (little zigzags, like a Z)
    // flying off its tip, then raised
    // straight up (its tip then sits at 55, 28) and the bolt comes down on it
    // with a flash. Afterwards smoke rises from the
    // hat, and shaking off the soot sends dust flying.
    out += `<g class="zap-shake"><g class="zap-wand">${wand}</g></g>` +
      [[-5, -12, -15], [6, -10, 20], [1, -14, 0], [8, -6, 30], [-3, -9, -25], [4, -13, 10]].map(([dx, dy, tilt], i) =>
        `<g class="zap-spark" style="--dx:${dx}px;--dy:${dy}px;animation-delay:${i / 10}s">` +
        `<path d="M66.6 27.2 H69.4 L67.2 30.2 H69.8 L67 33" transform="rotate(${tilt} 68 30)" fill="none" stroke="#FCE38A" stroke-width="1.1" stroke-linejoin="round" stroke-linecap="round"/></g>`).join("") +
      `<g class="zap-bolt"><path d="M63 0 L51 11 L57 12 L55 27" fill="none" stroke="#FCE38A" stroke-width="3.2" stroke-linejoin="round" stroke-linecap="round"/>` +
      `<path d="M63 0 L51 11 L57 12 L55 27" fill="none" stroke="#fff" stroke-width="1.2" stroke-linejoin="round" stroke-linecap="round"/></g>` +
      `<circle class="zap-flash" cx="55" cy="28" r="9" fill="#fff"/>` +
      [0, .25, .5].map(d => `<circle class="zap-smoke" style="animation-delay:${d}s" cx="36" cy="${3 + (shape.hatDy || 0)}" r="2.6" fill="#8a8780"/>`).join("") +
      [[-1, -.2], [1, -.3], [-1, .5], [1, .4], [-.5, -1], [.5, -1]].map(([ux, uy]) => {
        const reach = Math.min(shape.gap + 6, 16);
        return `<circle class="zap-dust" style="--dx:${ux * 10}px;--dy:${uy * 8}px" cx="${30 + ux * reach}" cy="${y + 8 + uy * 6}" r="1.3" fill="#5b5651"/>`;
      }).join("");
  } else if (status === "gust") {
    // The wand is waved about while a breeze blows by its tip (short lines
    // sliding from left to right, one after another), then a gust (streaks
    // with curled ends) sweeps across from the left and takes the hat. The
    // wand runs with the body.
    out += `<g class="gust-run"><g class="gust-wand">${wand}</g>` +
      [[61, 27, 7], [64, 33, 6], [60, 38, 7]].map(([x, y, w], i) =>
        `<path class="gust-breeze" style="animation-delay:${i * 150}ms" d="M${x} ${y} h${w}" stroke="#d6ecf5" stroke-width="1.3" stroke-linecap="round"/>`).join("") +
      `</g>` +
      `<g class="gust-hat" style="--hat-dy:${shape.hatDy || 0}px">${drawHat()}</g>` +
      [6, 17, 28].map((y, i) =>
        `<path class="gust-streak" style="animation-delay:${i * 60}ms" d="M-6 ${y} H18 q6 0 6 -4 q0 -3 -3 -3" fill="none" stroke="#d6ecf5" stroke-width="1.8" stroke-linecap="round"/>`).join("");
  } else if (status === "blow" || status === "blowswap") {
    // The gust's wand and breeze, once through, with no hat flying off here
    // (it flies with flyHats).
    out += `<g class="blow-wand">${wand}</g>` +
      [[61, 27, 7], [64, 33, 6], [60, 38, 7]].map(([x, y, w], i) =>
        `<path class="blow-breeze" style="animation-delay:${i * 150}ms" d="M${x} ${y} h${w}" stroke="#d6ecf5" stroke-width="1.3" stroke-linecap="round"/>`).join("");
  } else if (status === "hatswap") {
    // The gust sweeps across from the side the prankster is on.
    const streaks = [6, 17, 28].map((y, i) =>
      `<path class="swap-streak" style="animation-delay:${i * 60}ms" d="M-6 ${y} H18 q6 0 6 -4 q0 -3 -3 -3" fill="none" stroke="#d6ecf5" stroke-width="1.8" stroke-linecap="round"/>`).join("");
    out += `<g class="swap-hop">${wand}</g>` + ((opts.side || 1) > 0 ? `<g transform="matrix(-1 0 0 1 60 0)">${streaks}</g>` : streaks);
  } else if (status === "fire" || status === "fireprank") {
    // Fiery sparks fly off the waved wand, then it aims a little higher (tip
    // at 63.5, 30) and a fireball flies off to the top right. A spark from it
    // drops onto the middle of the hat (29, 15), where a flame grows slowly,
    // with smoke, and the hat chars. Later the wand points at the hat (tip at
    // 47, 29.6) and water drops put the flame out in a puff of steam. All
    // that's on the hat jumps with the body. As a prank ("fireprank") only
    // the wand, the sparks and the fireball, shot straight up (the wand's tip
    // at 55, 28) off the screen: later it comes down in the campfire
    // (dropFireball).
    const prank = status === "fireprank", pre = prank ? "fp" : "fire";
    const hd = shape.hatDy || 0;
    const flame = (w, h, fill, cls) => `<path class="${cls}" d="M${29 - w} 15 C${29 - w} ${15 - h * .45} ${29 - w * .3} ${15 - h * .65} 29 ${15 - h} ` +
      `C${29 + w * .3} ${15 - h * .65} ${29 + w} ${15 - h * .45} ${29 + w} 15 Q29 ${15 + w * 1.1} ${29 - w} 15Z" fill="${fill}"/>`;
    const [bx, by] = prank ? [55, 28] : [63.5, 30];
    const ball = (r, cls, delay) => `<g class="${cls}" style="animation-delay:${delay}ms">` +
      `<circle cx="${bx}" cy="${by}" r="${r}" fill="#E8572A"/><circle cx="${bx}" cy="${by}" r="${r * .6}" fill="#F59A3A"/><circle cx="${bx}" cy="${by - .5}" r="${r * .3}" fill="#FCE38A"/></g>`;
    const shot = [[-5, -12, 0], [6, -10, 1], [1, -14, 2], [8, -6, 0], [-3, -9, 1], [4, -13, 2]].map(([dx, dy, c], i) =>
        `<circle class="${pre}-spark" style="--dx:${dx}px;--dy:${dy}px;animation-delay:${i / 10}s" cx="67" cy="30" r="1.2" fill="${["#FCE38A", "#F59A3A", "#E8572A"][c]}"/>`).join("") +
      ball(2.4, `${pre}-trail`, 140) + ball(3, `${pre}-trail`, 70) + ball(4, `${pre}-ball`, 0);
    if (prank) out += `<g class="fp-wand">${wand}</g>` + shot;
    else out += `<g class="fire-jump"><g class="fire-wand">${wand}</g>` +
      [0, 1, 2, 3, 4, 5, 6].map(i => `<circle class="fire-water" style="animation-delay:${i * 100}ms" cx="47" cy="29.6" r="1.1" fill="${MAGIC["*"]}"/>`).join("") +
      `<g transform="translate(0 ${hd})">` +
      `<path class="fire-char" d="M24.5 15 C24 11.5 26.5 9.5 29 10.5 C31.5 9.2 34.2 11.2 33.6 14.2 C34.4 16.4 31.6 18 29.4 17 C27 18.2 24.2 17 24.5 15Z" fill="#2a2420"/>` +
      `<g class="fire-hat">${flame(4, 12, "#E8572A", "fire-flick-a")}${flame(2.2, 7, "#FCE38A", "fire-flick-c")}</g>` +
      [0, 1, 2, 3, 4, 5, 6, 7].map(i => `<circle class="fire-smoke" style="animation-delay:${i * 400}ms" cx="29" cy="3" r="1.8" fill="#8a8780"/>`).join("") +
      [-1, 0, 1].map(ux => `<circle class="fire-steam" style="--dx:${ux * 5}px" cx="${29 + ux * 2}" cy="6" r="2.6" fill="#e4e1da"/>`).join("") +
      `</g></g>` + shot +
      `<circle class="fire-drop" cx="29" cy="${14 + hd}" r="1.3" fill="#FCE38A"/>`;
  } else if (status === "soaked") {
    // The rain comes down on it, and twice it puffs a "hmph" at the prankster.
    out += wand + rainOn("soak") +
      [0, 1].map(i => `<g class="soak-puff" style="--side:${opts.side || 0};animation-delay:${i * 500}ms">${puff(opts.side || 1)}</g>`).join("");
  } else if (status === "rain" || status === "prank") {
    // Droplets fly off the waved wand, then it aims a little higher and a
    // cartoon stream of water (see RAIN_ARC) shoots off to the top right,
    // past the edge of the screen, with a splash at its head: it grows out of
    // the wand and later lets go of it. Then the rain falls from above (on
    // the prankster's friends instead, see "soaked"; then it raises the wand
    // upright and the stream shoots straight up, RAIN_UP); drips run off the
    // soaked body, and steam rises as it dries.
    const prank = status === "prank";
    const WATER = [MAGIC["*"], "#b8ecff"];
    const P = prank ? RAIN_UP : RAIN_ARC, OUTLINE = "#1b6fae", BLUE = "#49b9ef", DEEP = "#2381c9", FOAM = "#e8f8ff";
    const arc = `M${P[0]} C${P[1]} ${P[2]} ${P[3]}`;
    // The stream: an outlined band of blue, then bright glints and darker
    // patches flowing along it while it pours.
    const stream = (color, width) => `<path class="rain-stream" d="${arc}" pathLength="100" fill="none" stroke="${color}" stroke-width="${width}" stroke-linecap="round"/>`;
    const flow = (cls, color, width) => `<path class="rain-flow ${cls}" d="${arc}" pathLength="100" fill="none" stroke="${color}" stroke-width="${width}" stroke-linecap="round"/>`;
    // The splash at the head (see addRainKeyframes), drawn pointing along +x:
    // a spiky burst with a dark heart and a few droplets flying ahead.
    const burst = Array.from({ length: 16 }, (_, i) => {
      const a = -Math.PI + i * Math.PI / 8, front = Math.cos(a) > -.2;
      const r = !front ? 2.6 : i % 2 ? 2.8 : 4.8 + (i % 4 === 0 ? 1.4 : 0);
      return `${i ? "L" : "M"}${(r * Math.cos(a)).toFixed(2)} ${(r * Math.sin(a)).toFixed(2)}`;
    }).join(" ") + "Z";
    const head = `<g class="${prank ? "prank-head" : "rain-head"}"><g class="rain-head-pulse"><g transform="scale(1.4)">` +
      `<path d="${burst}" fill="${BLUE}" stroke="${OUTLINE}" stroke-width=".8" stroke-linejoin="round"/>` +
      `<ellipse cx=".6" cy="0" rx="1.6" ry="1.3" fill="${DEEP}"/>` +
      [[6.5, -3.5, .8], [7.5, 2.5, .7], [5, 5, .6], [8.5, -.5, .6], [4.5, -5.5, .5]].map(([x, y, r]) =>
        `<circle cx="${x}" cy="${y}" r="${r}" fill="${BLUE}" stroke="${OUTLINE}" stroke-width=".4"/>`).join("") +
      `</g></g></g>`;
    // While it pours, droplets keep splashing off the stream on both sides:
    // each starts at a point along the arc (t) and flies off across it,
    // falling a little, over and over (rain-splash) with its own delay.
    const splash = Array.from({ length: 24 }, (_, i) => {
      const t = .03 + (i / 24) * .7, side = i % 2 ? 1 : -1;
      const tx = bezierSlope(P, t, 0), ty = bezierSlope(P, t, 1), len = Math.hypot(tx, ty);
      const reach = 4 + (i * 7 % 5) * 1.1, ahead = (i * 3 % 5) * 1.2;
      const dx = (-ty / len) * side * reach + (tx / len) * ahead, dy = (tx / len) * side * reach + (ty / len) * ahead + 4;
      return `<circle class="rain-splash" style="--dx:${dx.toFixed(1)}px;--dy:${dy.toFixed(1)}px;animation-delay:${-(i * 137 % 600)}ms" ` +
        `cx="${bezier(P, t, 0).toFixed(1)}" cy="${bezier(P, t, 1).toFixed(1)}" r="${.6 + (i % 3) * .25}" fill="${BLUE}"/>`;
    }).join("");
    out += `<g class="${prank ? "prank-shake" : ""}"><g class="${prank ? "prank-wand" : "rain-wand"}">${wand}</g></g>` +
      [[-5, -12], [6, -10], [1, -14], [8, -6], [-3, -9], [4, -13]].map(([dx, dy], i) =>
        `<circle class="rain-spark" style="--dx:${dx}px;--dy:${dy}px;animation-delay:${i / 10}s" cx="67" cy="30" r="1.1" fill="${WATER[i % 2]}"/>`).join("") +
      `<g class="rain-spray">${splash}</g>` +
      stream(OUTLINE, 6.2) + stream(BLUE, 4.4) +
      flow("rain-patch", DEEP, 3) + flow("rain-glint", FOAM, 1.1) + head +
      (prank ? "" : rainOn("rain"));
  } else if (!sulky) {
    // Waving to a friend (see life.js) it swings the wand and calls "hey, hey!".
    out += status === "waving"
      ? `<g class="wand-wave">${wand}</g><ellipse class="call" cx="30" cy="${shape.eyeY + 7}" rx="2.6" ry="2.4" fill="${EYE}"/>`
      : wand;
  }
  if (changing) {
    // Ease back from wherever the hop was when the change began.
    const from = `--from-y:${change.fromY || 0}px;--from-o:${change.fromOpacity ?? 1};animation-delay:${-change.ago}ms`;
    return `<svg class="mascot changing" width="62" height="48" viewBox="0 0 78 60" aria-hidden="true" style="${from}">${out}</svg>`;
  }
  if (status === "working") {
    // Sparkles fly off the wand in two frames, one on the way up, one on the way down.
    const frames = [[[71, 22, 3], [74, 9, 2.5]], [[75, 20, 2.5], [65, 16, 3], [70, 4, 2.5]]];
    out += frames.map((pts, i) =>
      `<g class="fr" style="animation-delay:${clock(HOP) - (HOP / 2) * i}ms">${pts.map(([x, y, r]) => sparkle(x, y, r, MAGIC["*"])).join("")}</g>`).join("");
  }
  if (asleep) {
    // Three Zs rise from the hat one after another, one per breath, growing and fading out.
    out += [0, 1, 2].map(i =>
      `<text class="zzz" x="58" y="26" font-size="11" font-weight="700" fill="#b4b2a9" font-family="system-ui, sans-serif" style="animation-delay:${clock(SNORE) - (SNORE / 3) * i}ms">Z</text>`).join("");
  }
  // The hop runs on the whole mascot; timing it by the clock too keeps it
  // smooth when the card is redrawn.
  const whole = status === "working" || status === "waiting" ? clock(HOP) : null;
  const style = whole === null ? "" : ` style="animation-delay:${whole}ms"`;
  return `<svg class="mascot" width="62" height="48" viewBox="0 0 78 60" aria-hidden="true"${style}>${out}</svg>`;
}

// flyHats flies the two hats of a hat swap (see SWAP) over layer, starting
// in ms from now: pair is the two mascots, each { index, box, side } with
// box its svg's { x, y, w, h } in the layer's px and side where the other
// one is. Each hat spins up from its owner's head, crosses over and lands
// crooked on the other's head, as the other draws it ("hatswap"); later it's
// tossed back. Between flights the mascots draw the hats themselves.
function flyHats(layer, pair, start) {
  pair.forEach((o, i) => {
    const w = pair[1 - i], s = o.box.w / 78;
    const dyO = SHAPES[o.index % SHAPES.length].hatDy || 0, dyW = SHAPES[w.index % SHAPES.length].hatDy || 0;
    const el = document.createElement("div");
    el.style.cssText = `position:absolute;left:${o.box.x}px;top:${o.box.y}px;width:${o.box.w}px;height:${o.box.h}px;` +
      `opacity:0;pointer-events:none;transform-origin:${30 * s}px ${(16 + dyO) * s}px`;
    el.innerHTML = `<svg viewBox="0 0 78 60" width="100%" height="100%" style="overflow:visible">${hatSVG(HATS[o.index % HATS.length], dyO)}</svg>`;
    layer.appendChild(el);
    // Where it lands: on the other's head, shifted and turned as worn there.
    const tx = w.box.x - o.box.x - w.side * 2 * s, ty = w.box.y - o.box.y + (dyW - dyO) * s, tilt = -w.side * SWAP_TILT;
    const peak = (i ? 42 : 58) * s; // one flies higher, so they pass each other
    const at = (x, y, turn) => ({ opacity: 1, transform: `translate(${x}px, ${y}px) rotate(${turn}deg)` });
    el.animate([at(0, 0, 0), { ...at(tx * .2, -peak * .75, 200), offset: .3 }, { ...at(tx * .5, -peak, 400), offset: .55 },
      { ...at(tx * .85, ty - peak * .5, 620), offset: .8 }, at(tx, ty, 720 + tilt)],
      { duration: SWAP_FLY[1] - SWAP_FLY[0], delay: start + SWAP_FLY[0], easing: "ease-in-out" });
    el.animate([at(tx, ty, tilt), at(tx * .5, Math.min(ty, 0) - 28 * s, -180), at(0, 0, -360)],
      { duration: SWAP_BACK[1] - SWAP_BACK[0], delay: start + SWAP_BACK[0], easing: "ease-in-out" });
    setTimeout(() => el.remove(), start + SWAP_BACK[1] + 100);
  });
}

// dropFireball drops a fire prank's fireball (see BLAZE_AT) into a campfire
// over layer, landing in ms from now: fire is { el, x, y, w, h }, the
// campfire's element and its svg's box in the layer's px. The ball falls
// from above the screen, and the fire flares up with a burst of light and
// embers.
function dropFireball(layer, fire, start) {
  const DROP = 600, FLARE = 2200;
  const cx = fire.x + fire.w / 2, cy = fire.y + fire.h * .7, r = fire.h * .14;
  const add = (css, html = "") => {
    const el = document.createElement("div");
    el.style.cssText = `position:absolute;opacity:0;pointer-events:none;${css}`;
    el.innerHTML = html;
    layer.appendChild(el);
    setTimeout(() => el.remove(), start + FLARE + 100);
    return el;
  };
  const ball = add(`left:${cx - r}px;top:${cy - r}px;width:${2 * r}px;height:${2 * r}px`,
    `<svg viewBox="-10 -10 20 20" width="100%" height="100%"><circle r="10" fill="#E8572A"/><circle r="6" fill="#F59A3A"/><circle r="3" cy="-1" fill="#FCE38A"/></svg>`);
  ball.animate([{ opacity: 1, transform: `translateY(${-(cy + r + 40)}px)` }, { opacity: 1, transform: "translateY(0)" }],
    { duration: DROP, delay: start - DROP, easing: "ease-in" });
  const glow = add(`left:${cx - fire.h * 3}px;top:${cy - fire.h * 3}px;width:${fire.h * 6}px;height:${fire.h * 6}px;border-radius:50%;` +
    `background:radial-gradient(closest-side, rgba(255, 190, 90, .55), rgba(242, 120, 40, .18) 55%, transparent)`);
  layer.prepend(glow); // behind the mascots
  glow.animate([{ opacity: 0, transform: "scale(.4)" }, { opacity: 1, transform: "scale(1)", offset: .12 }, { opacity: 0, transform: "scale(1.15)" }],
    { duration: FLARE, delay: start, easing: "ease-out" });
  for (let i = 0; i < 10; i++) {
    const a = -Math.PI * (.1 + .8 * i / 9), d = fire.h * (.8 + (i % 3) * .3);
    const ember = add(`left:${cx - 2}px;top:${cy - 2}px;width:4px;height:4px;border-radius:50%;background:${i % 2 ? "#FCE38A" : "#F59A3A"}`);
    ember.animate([{ opacity: 1, transform: "translate(0, 0)" }, { opacity: 0, transform: `translate(${Math.cos(a) * d}px, ${Math.sin(a) * d}px)` }],
      { duration: 900, delay: start, easing: "ease-out" });
  }
  fire.el.firstElementChild.style.transformOrigin = "50% 90%";
  fire.el.firstElementChild.animate([{ transform: "scale(1)", filter: "brightness(1)" },
    { transform: "scale(1.9)", filter: "brightness(1.6)", offset: .12 }, { transform: "scale(1.5)", filter: "brightness(1.3)", offset: .45 },
    { transform: "scale(1)", filter: "brightness(1)" }], { duration: FLARE, delay: start, easing: "ease-out" });
}

// When agents of one project are all asleep they rest together by a campfire.
// One flicker of the flames takes FLICKER, an ember rises over EMBER. Keep in
// step with flicker-* and ember in mascot.css.
const FLICKER = 900;
const EMBER = 2400;

// campfireSVG draws the fire, on the same 60-high canvas as the mascot so they
// stand on one ground line: crossed logs, three layers of flame and embers.
function campfireSVG() {
  const d = period => `animation-delay:${-(Date.now() % period)}ms`;
  const flame = (cls, color, w, top) =>
    `<path class="${cls}" style="${d(FLICKER)}" d="M${22 - w} 52 C${22 - w} 44 ${22 - w / 3} 40 22 ${top} C${22 + w / 3} 40 ${22 + w} 44 ${22 + w} 52Z" fill="${color}"/>`;
  return `<svg class="campfire-svg" width="44" height="60" viewBox="0 0 44 60" aria-hidden="true">` +
    `<ellipse class="glow" style="${d(FLICKER)}" cx="22" cy="55" rx="21" ry="4" fill="#F2A03D" opacity=".25"/>` +
    flame("flicker-a", "#E8572A", 10, 24) + flame("flicker-b", "#F59A3A", 7, 32) + flame("flicker-c", "#FCE38A", 3.5, 41) +
    `<rect x="7" y="50" width="30" height="5" rx="2.5" fill="#7A4A23" transform="rotate(12 22 52)"/>` +
    `<rect x="7" y="50" width="30" height="5" rx="2.5" fill="#8A5A2B" transform="rotate(-12 22 52)"/>` +
    [[18, 0], [25, EMBER / 3], [21, EMBER * 2 / 3]].map(([x, off]) =>
      `<circle class="ember" style="animation-delay:${-((Date.now() + off) % EMBER)}ms" cx="${x}" cy="36" r="1.2" fill="#FCE38A"/>`).join("") +
    `</svg>`;
}
