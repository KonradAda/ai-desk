// The agent mascot: a creature in a wizard hat holding a wand, drawn as SVG.
// Shared by the simulator (index.html) and the animation gallery (gallery.html);
// its animations are in mascot.css.

const COLORS = ["#D85A30", "#1D9E75", "#7F77DD", "#EF9F27", "#378ADD", "#D4537E", "#97C459", "#5DCAA5"];
const EYE = "#121211";
// H hat, Y gold, B wood, * sparkle.
const MAGIC = { H: "#534AB7", Y: "#F2C14E", B: "#8A5A2B", "*": "#7FD6FF" };
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

// A four-point sparkle centred on x, y.
const sparkle = (x, y, r, fill) =>
  `<path d="M${x} ${y-r} Q${x} ${y} ${x+r} ${y} Q${x} ${y} ${x} ${y+r} Q${x} ${y} ${x-r} ${y} Q${x} ${y} ${x} ${y-r}Z" fill="${fill}"/>`;
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

function mascotSVG(index, status, change = null) {
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
  // The purple mascot would blend into the purple hat, so it gets a burgundy one.
  const hat = color === "#7F77DD" ? "#993556" : MAGIC.H;

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

  // drawBody draws the body with its extras, face and hat. eyesWrap, when
  // set, wraps the eyes (e.g. in a blinking group).
  const drawBody = (eyes, eyesWrap = e => e) => {
    const cheekX = shape.cheekX ?? shape.gap + 7;
    return (shape.feet ? [18, 36].map(x => `<rect x="${x}" y="52" width="6" height="7" rx="3" fill="${color}"/><rect x="${x}" y="52" width="6" height="7" rx="3" fill="#000" opacity=".18"/>`).join("") : "") +
      (shape.behind ? shape.behind(color) : "") +
      `<path d="${shape.path}" fill="${color}"/><path d="${shape.path}" fill="url(#mascot-shade)"/>` +
      [30 - cheekX, 30 + cheekX].map(x => `<ellipse cx="${x}" cy="${shape.eyeY+6}" rx="3.2" ry="2" fill="#fff" opacity=".18"/>`).join("") +
      eyesWrap(drawEyes(eyes)) +
      `<g transform="translate(0 ${shape.hatDy || 0})">` +
      `<path d="M14 26 C21 21 25 11 36 2 C34 11 38 19 46 26Z" fill="${hat}"/>` +
      `<ellipse cx="30" cy="27" rx="22" ry="4" fill="${hat}"/>` +
      `<ellipse cx="30" cy="26" rx="22" ry="3" fill="#000" opacity=".15"/>` +
      sparkle(31, 17, 3.5, MAGIC.Y) + `</g>`;
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
  if (status === "waiting" && !changing) {
    // A sign on a stick instead of the wand, waved from where the stick is held.
    out += `<g class="sign" style="animation-delay:${clock(WAVE)}ms">` +
      `<line x1="56" y1="50" x2="64" y2="22" stroke="${MAGIC.B}" stroke-width="3" stroke-linecap="round"/>` +
      `<rect x="54" y="4" width="22" height="19" rx="3" fill="#f1efe8" stroke="#000" stroke-opacity=".25" stroke-width="1.2"/>` +
      `<text x="65" y="19.5" text-anchor="middle" font-size="16" font-weight="800" fill="${EYE}" font-family="system-ui, sans-serif">!</text></g>` +
      // With each wave the mouth opens twice, as if it calls "hey, hey!".
      `<ellipse class="call" cx="30" cy="${shape.eyeY + 7}" rx="2.6" ry="2.4" fill="${EYE}" style="animation-delay:${clock(WAVE)}ms"/>`;
  } else if (!sulky) {
    // Waving to a friend (see life.js) it swings the wand and calls "hey, hey!".
    const wand = `<line x1="55" y1="48" x2="66" y2="34" stroke="${MAGIC.B}" stroke-width="3" stroke-linecap="round"/>` + sparkle(67, 32, 3.5, MAGIC.Y);
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
