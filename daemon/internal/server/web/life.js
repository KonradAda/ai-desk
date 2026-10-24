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

const LOOK_MS = 2400; // looking around after finishing; keep in step with look in mascot.css
const WAVE_MS = 1300; // waving at a friend; keep in step with wand-wave in mascot.css
const WALK_SPEED = 0.07; // px per ms on the 800x480 screen

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
// Walks last life[id].walkMs, which the screen sets from the distance.
const CAMPING = ["wave", "rouse", "out", "settle", "camp"];
const AT_FIRE = ["settle", "camp"];

const isAsleep = status => status === "done" || status === "idle";

// stepLife moves every agent's life on to now. runs are the groups on the
// screen: [{ key, sessions }], sessions in card order. life (id -> { phase,
// at, walkMs }) is updated in place; an agent seen for the first time is put
// straight where it belongs (asleep agents of one group already by the fire),
// with no animation.
function stepLife(life, runs, now) {
  const first = Object.keys(life).length === 0;
  const go = (e, phase) => { e.phase = phase; e.at = now; delete e.walkMs; };
  for (const run of runs) {
    const fresh = run.sessions.filter(s => !life[s.id]);
    const freshAsleep = fresh.filter(s => isAsleep(s.status));
    for (const s of fresh) {
      life[s.id] = { phase: !isAsleep(s.status) ? "home" : first && freshAsleep.length > 1 ? "camp" : "sleep", at: now - 1e6 };
    }
    const entry = s => life[s.id];
    for (const s of run.sessions) {
      const e = entry(s), since = now - e.at;
      if (!isAsleep(s.status)) {
        // Work (or a question) calls it home: from the fire it walks back.
        if (CAMPING.includes(e.phase)) go(e, "back");
        else if (e.phase !== "home" && e.phase !== "back") go(e, "home");
        else if (e.phase === "back" && e.walkMs !== undefined && since >= e.walkMs) go(e, "home");
        continue;
      }
      switch (e.phase) {
        case "home": go(e, "look"); break;
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
