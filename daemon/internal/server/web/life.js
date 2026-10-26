// The mascots' life between jobs, as a small state machine with no drawing
// in it (the screen draws and moves the mascots; later the firmware does the
// same from this logic).
//
// An agent that finishes wakes up in its card and looks around. If nobody of
// its group (the agents of one project side by side in a row) is free, it
// gets sleepy and falls asleep in its card. If a friend is free, it waves; a
// friend who was asleep wakes up; both walk to the middle, where a campfire
// lights up, sit down and fall asleep together. One that gets work walks back
// to its card; one left alone by the fire walks back too and sleeps there.
//
// At work, now and then, it puts on a little show: it always reads its
// spellbook first, then casts one of the SPELLS (mascot.js), picked at random.
// Rain may turn into a prank: with friends of its group asleep, it's a toss
// of a coin whether the rain comes down on it or on some of them (how many
// and who, at random too), who wake up soaked and cross while it laughs.
// The gust may too: it swaps the hats of two friends asleep (picked at
// random), or with only one asleep, that one's and its own. And the fire:
// with friends asleep by the campfire, the fireball may come down in the
// fire, which flares up and startles them, and it's ashamed.

const LOOK_MS = 2400; // looking around after finishing; keep in step with look in mascot.css
const WAVE_MS = 1300; // waving at a friend; keep in step with wand-wave in mascot.css
const WALK_SPEED = 0.07; // px per ms on the 800x480 screen
const SHOW_GAP = [5000, 20000]; // a working mascot's shows come this far apart, at random

// Phases (life[id].phase), each since life[id].at:
//   home    in its card, drawn by its status (working, waiting, ...)
//   look    just finished, awake in its card, looking around
//   doze    nobody around: falls asleep in its card (the sleep transition)
//   sleep   asleep in its card
//   wave    waves at a friend before setting off to the fire
//   rouse   woken by a friend's wave (the wake transition), then sets off
//   out     walking to the fire
//   settle  arrived at the fire, falls asleep (the sleep transition)
//   camp    asleep by the fire
//   back    walking back to its card
//   read    at work: reads its spellbook (READ)
//   spell   at work: casts life[id].spell, one of the SPELLS
// Walks last life[id].walkMs, which the screen sets from the distance.
// Between shows life[id].nextShow says when the next one starts. A rain cast
// as a prank lists its victims in life[id].targets; a victim asleep in its
// card or by the fire has life[id].soak = { at, by }: from at it's soaked for
// SOAK (mascot.js), looking towards by, the prankster. A gust cast as a
// prank keeps its hat swap in life[id].swap = { at, pair, flown } (pair the
// two ids whose hats swap at at, flown once the screen has sent the hats
// flying) and lasts life[id].showMs; each sleeper in the swap has
// life[id].swap = { at, other }, the one it swaps with, for SWAP from at.
// A fire cast as a prank keeps life[id].blaze = { at, flown } (the fireball
// lands in the campfire at at); each one asleep by the fire then has
// life[id].startle = { at }, startled for STARTLE from at.
const CAMPING = ["wave", "rouse", "out", "settle", "camp"];
const AT_FIRE = ["settle", "camp"];
const SOAKABLE = ["sleep", "camp"]; // asleep and staying put: in its card or by the fire

const isAsleep = status => status === "done" || status === "idle";

// A session that changed less than this long ago (by the daemon's clock) is
// new news: seen for the first time and free, it looks around like one that
// just finished.
const RECENT_MS = 15000;

// stepLife moves every agent's life on to now. runs are the groups on the
// screen: [{ key, sessions }], sessions in card order; serverNow is now by
// the daemon's clock. life (id -> { phase, at, walkMs }) is updated in place.
// An agent seen for the first time that just became free (a new session with
// nothing to do yet) looks around and may join the others; one that has been
// asleep a while (the screen was opened, the page turned) is put straight
// where it belongs, by the fire with the others or in its card, with no
// animation.
function stepLife(life, runs, now, serverNow) {
  const go = (e, phase) => { e.phase = phase; e.at = now; delete e.walkMs; };
  for (const run of runs) {
    const fresh = run.sessions.filter(s => !life[s.id]);
    const settled = fresh.filter(s => isAsleep(s.status) && serverNow - Date.parse(s.updated_at) >= RECENT_MS);
    for (const s of fresh) {
      life[s.id] = { phase: !settled.includes(s) ? "home" : settled.length > 1 ? "camp" : "sleep", at: now - 1e6 };
    }
    const entry = s => life[s.id];
    for (const s of run.sessions) {
      const e = entry(s), since = now - e.at;
      // A soaking lasts while it sleeps on where it is, and no longer than SOAK.
      if (e.soak && (!isAsleep(s.status) || !SOAKABLE.includes(e.phase) || now >= e.soak.at + SOAK)) delete e.soak;
      // So does a hat swap.
      if (e.swap?.other && (!isAsleep(s.status) || !SOAKABLE.includes(e.phase) || now >= e.swap.at + SWAP)) delete e.swap;
      if (e.startle && (!isAsleep(s.status) || e.phase !== "camp" || now >= e.startle.at + STARTLE)) delete e.startle;
      if (!isAsleep(s.status)) {
        // Work (or a question) calls it home: from the fire it walks back.
        if (CAMPING.includes(e.phase)) go(e, "back");
        else if (e.phase === "read" || e.phase === "spell") stepShow(e, s, since, now, run, life);
        else if (e.phase !== "home" && e.phase !== "back") go(e, "home");
        else if (e.phase === "back" && e.walkMs !== undefined && since >= e.walkMs) go(e, "home");
        else if (e.phase === "home") waitForShow(e, s, now);
        continue;
      }
      switch (e.phase) {
        case "home": case "read": case "spell": go(e, "look"); break;
        case "look": {
          if (since < LOOK_MS) break;
          // Anyone else of the group free (looking around, asleep or by the
          // fire) is company: wave and gather them.
          const friends = run.sessions.filter(f => f !== s && isAsleep(f.status) && entry(f).phase !== "home");
          if (!friends.length) { go(e, "doze"); break; }
          go(e, "wave");
          for (const f of friends) {
            const fe = entry(f);
            if (fe.phase === "doze" || fe.phase === "sleep") go(fe, "rouse");
            else if (fe.phase === "look") go(fe, "wave");
            else if (fe.phase === "back") go(fe, "out");
          }
          break;
        }
        case "wave": if (since >= WAVE_MS) go(e, "out"); break;
        case "rouse": if (since >= TRANSITIONS.wake.ms) go(e, "out"); break;
        case "out": if (e.walkMs !== undefined && since >= e.walkMs) go(e, "settle"); break;
        case "settle": if (since >= TRANSITIONS.sleep.ms) go(e, "camp"); break;
        case "doze": if (since >= TRANSITIONS.sleep.ms) go(e, "sleep"); break;
        case "back": if (e.walkMs !== undefined && since >= e.walkMs) go(e, "doze"); break;
      }
    }
    // Nobody to sit with any more: the last one by the fire goes back to its card.
    const campers = run.sessions.filter(s => CAMPING.includes(entry(s).phase));
    if (campers.length === 1) go(entry(campers[0]), "back");
  }
}

// waitForShow counts down to a working mascot's next show and starts it
// with the spellbook; the countdown starts afresh with each stretch of work.
function waitForShow(e, s, now) {
  if (s.status !== "working") { delete e.nextShow; return; }
  if (e.nextShow === undefined) e.nextShow = now + SHOW_GAP[0] + Math.random() * (SHOW_GAP[1] - SHOW_GAP[0]);
  else if (now >= e.nextShow) Object.assign(e, { phase: "read", at: now });
}

// stepShow plays a show on: the spellbook, then a spell picked at random
// (not the last one again, when there are others), then back to work. A
// question or an interruption cuts it short, and a prank's rain or hat swap
// that hasn't started yet doesn't.
function stepShow(e, s, since, now, run, life) {
  const end = () => {
    for (const id of e.targets || []) if (life[id]?.soak?.at > now) delete life[id].soak;
    if (e.swap?.at > now) for (const id of e.swap.pair) if (id !== s.id && life[id]?.swap?.at === e.swap.at) delete life[id].swap;
    if (e.blaze?.at > now) for (const f of run.sessions) if (life[f.id]?.startle?.at === e.blaze.at) delete life[f.id].startle;
    delete e.targets; delete e.swap; delete e.showMs; delete e.blaze;
    Object.assign(e, { phase: "home", at: now });
  };
  if (s.status !== "working") return end();
  if (e.phase === "read" && since >= READ) {
    const names = Object.keys(SPELLS), fresh = names.filter(n => n !== e.spell);
    const pick = fresh.length ? fresh : names;
    Object.assign(e, { phase: "spell", at: now, spell: pick[Math.floor(Math.random() * pick.length)] });
    if (e.spell === "rain") prank(e, s, run, life, now);
    if (e.spell === "gust") swapPrank(e, s, run, life, now);
    if (e.spell === "fire") firePrank(e, s, run, life, now);
  } else if (e.phase === "spell" && since >= (e.showMs ?? SPELLS[e.spell])) {
    end();
    delete e.nextShow;
  }
}

// sleepersOf: the friends of the group asleep in their cards or by the fire,
// in random order.
const sleepersOf = (s, run, life) => run.sessions
  .filter(f => f !== s && isAsleep(f.status) && SOAKABLE.includes(life[f.id]?.phase))
  .map(f => [Math.random(), f]).sort((a, b) => a[0] - b[0]).map(([, f]) => f);

// prank: with friends of the group asleep, half the time the rain comes down
// on some of them instead: one or more, as many and whichever chance picks.
function prank(e, s, run, life, now) {
  const sleepers = sleepersOf(s, run, life);
  if (!sleepers.length || Math.random() < .5) return;
  e.targets = sleepers.slice(0, 1 + Math.floor(Math.random() * sleepers.length)).map(f => f.id);
  for (const id of e.targets) life[id].soak = { at: now + PRANK_AT, by: s.id };
}

// swapPrank: with friends of the group asleep, half the time the gust swaps
// the hats of two of them, whichever chance picks, or with only one asleep,
// that one's and its own (and then the show lasts until its hat is back).
function swapPrank(e, s, run, life, now) {
  const sleepers = sleepersOf(s, run, life);
  if (!sleepers.length || Math.random() < .5) return;
  const pair = sleepers.length > 1 ? [sleepers[0].id, sleepers[1].id] : [s.id, sleepers[0].id];
  const at = now + SWAP_AT;
  e.swap = { at, pair, flown: false };
  if (pair.includes(s.id)) e.showMs = SWAP_AT + SWAP;
  pair.forEach((id, i) => { if (id !== s.id) life[id].swap = { at, other: pair[1 - i] }; });
}

// firePrank: with friends of the group asleep by the campfire, half the time
// the fireball goes straight up instead and comes down in the fire a while
// later, startling them all (and the show lasts FIRE_PRANK).
function firePrank(e, s, run, life, now) {
  const campers = run.sessions.filter(f => f !== s && isAsleep(f.status) && life[f.id]?.phase === "camp");
  if (!campers.length || Math.random() < .5) return;
  const at = now + BLAZE_AT;
  e.blaze = { at, flown: false };
  e.showMs = FIRE_PRANK;
  for (const f of campers) life[f.id].startle = { at };
}

// moveCamp: the spot by the fire moved (the cards changed, someone joined or
// left). Those sitting there wake up and walk over again; one on its way
// sets off anew from where it is.
function moveCamp(e, now) {
  if (AT_FIRE.includes(e.phase)) e.phase = "rouse";
  else if (e.phase !== "out") return;
  e.at = now;
  delete e.walkMs;
}
