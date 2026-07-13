# Survival — a line-by-line walkthrough

This document explains every line of `main.go`. It assumes you can program a little, but **not**
that you know Go or game programming. Where a line uses something Go-specific, the Go feature is
explained the first time it appears.

Line numbers match `main.go` exactly. If you edit the file, they drift — the section headings tell
you where you are regardless.

## How a game loop works (read this first)

The whole program is built around one idea. Ebitengine, the game library, calls two of our methods
over and over, forever:

- **`Update()`** — runs **60 times per second**, on a fixed schedule. One call is called a **tick**.
  It moves everything by one step: the player, the bullets, the enemies. It changes the game's
  state. It draws nothing.
- **`Draw()`** — runs once per frame, and paints the current state to the screen. It changes
  nothing.

That split is the single most important rule in the file. `Update` decides *what is true*; `Draw`
only *shows* what is already true. If you put game logic in `Draw`, it will run at a speed that
depends on the monitor, and the game will behave differently on different machines.

Everything below is in service of those two methods.

---

## Lines 1–2 — the package declaration

```go
1  // Command survival is a round-based survival shooter: WASD to move, mouse to aim, click to fire.
2  package main
```

**Line 1** is a comment. Go convention: a comment directly above `package` is the package's
documentation, and for a runnable program it starts with "Command <name>".

**Line 2** declares the package. `package main` is special in Go: it means "this code compiles into
a runnable program," not a library. A `main` package must contain a function called `main` (line
237) — that's where the program starts.

---

## Lines 4–16 — imports

```go
 4  import (
 5      "fmt"
 6      "image/color"
 7      "log"
 8      "math"
 9      "math/rand/v2"
10
11      "github.com/hajimehoshi/ebiten/v2"
12      "github.com/hajimehoshi/ebiten/v2/inpututil"
13      "github.com/hajimehoshi/ebiten/v2/text/v2"
14      "github.com/hajimehoshi/ebiten/v2/vector"
15      "golang.org/x/image/font/basicfont"
16  )
```

An `import` block lists the other packages this file uses. Go is strict: import something you don't
use and the program **will not compile**. That's deliberate — it stops dead imports accumulating.

- **Line 5** — `fmt`: formatting. Used only for `fmt.Sprintf`, which builds the HUD strings.
- **Line 6** — `image/color`: the standard `color.RGBA` type. Every colour in the game is one.
- **Line 7** — `log`: used once, on line 241, to report a fatal error and exit.
- **Line 8** — `math`: `math.Atan2`, `math.Hypot`, `math.Cos`, `math.Sin`, `math.Sqrt2`.
- **Line 9** — `math/rand/v2`: random numbers. The `/v2` matters: it's the modern generator, and it
  seeds itself. The old `math/rand` needed manual seeding or every run was identical.
- **Line 10** — blank. Convention: standard-library imports above, third-party below.
- **Line 11** — `ebiten`: the game engine. Window, GPU, input, the game loop itself.
- **Line 12** — `inpututil`: helpers on top of ebiten's input. We need exactly one — "was this key
  *just* pressed this tick," as opposed to "is it held down."
- **Line 13** — `text/v2`: drawing text.
- **Line 14** — `vector`: drawing shapes. We only ever draw filled circles.
- **Line 15** — `basicfont`: a tiny built-in bitmap font, so the game ships no font file.

---

## Lines 18–26 — the constants: every gameplay knob

```go
18  const (
19      screenW, screenH                        = 640, 480
20      playerRadius, playerSpeed, playerMaxHP  = 10, 3, 100
21      bulletRadius, bulletSpeed, fireCooldown = 3, 8, 8 // cooldown in ticks
22      contactDamage, roundHeal                = 12, 15
23      packRadius, packHeal, packScore         = 6, 20, 25
24      packChance                              = 4  // one kill in packChance drops a health pack
25      finalRound                              = 10 // clear this wave and the run is won
26  )
```

A `const` is a value fixed at compile time. Grouping them in one block at the top is a rule of this
project: **no magic numbers in the simulation code.** If you want to change how the game feels, you
change it here, and you never have to hunt through the logic.

- **Line 19** — the playing field is 640×480 *logical* pixels. "Logical" because the window can be
  resized; the game still thinks in 640×480 and Ebitengine scales it (see `Layout`, line 235).
- **Line 20** — the player is a circle of radius 10, moves 3 pixels per tick (so 180 pixels/second
  at 60 ticks), and starts with 100 health.
- **Line 21** — bullets are radius 3 and travel 8 pixels per tick. `fireCooldown = 8` means: after
  firing, wait 8 ticks before firing again. At 60 ticks/second that's 7.5 shots per second. The
  comment exists because "8" alone doesn't tell you the unit — ticks, not pixels, not seconds.
- **Line 22** — touching an enemy costs 12 health; surviving a round restores 15.
- **Line 23** — a health pack is a circle of radius 6, heals 20, and is worth 25 points.
- **Line 24** — `packChance = 4` means a 1-in-4 (25%) drop chance. See line 174 for the roll.
- **Line 25** — clear round 10 and you win.

Note that Go infers the type of each constant. `playerSpeed = 3` is an untyped constant, so it can
be used as a `float64` on line 135 without a conversion. That's a Go convenience you should not
expect from most languages.

---

## Lines 28–42 — package-level variables

```go
28  var (
29      face      = text.NewGoXFace(basicfont.Face7x13)
30      colBG     = color.RGBA{18, 18, 24, 255}
31      colPlayer = color.RGBA{80, 220, 120, 255}
32      colBullet = color.RGBA{250, 210, 90, 255}
33      colPack   = color.RGBA{110, 200, 255, 255}
```

These are `var`, not `const`, because Go only allows simple values (numbers, strings, booleans) to
be constants. A struct like `color.RGBA` has to be a variable.

- **Line 29** — builds the font once, at program start, and reuses it for every frame.
  `basicfont.Face7x13` is a 7×13-pixel bitmap font; `text.NewGoXFace` wraps it in the type
  Ebitengine's text package wants. Doing this once matters: building it per frame would be wasteful.
- **Lines 30–33** — the palette. `color.RGBA{R, G, B, A}` is red, green, blue, alpha (opacity),
  each 0–255. `255` alpha means fully opaque. So `colBG` is a very dark blue-grey, `colPlayer` a
  green, `colBullet` a yellow, `colPack` a light blue.

```go
35      // kinds are the enemy archetypes — steady grunt, fragile sprinter, slow brute — drawn at random
36      // per spawn and then scaled by the round in nextRound.
37      kinds = []ent{
38          {spd: 1.0, hp: 2, r: 9, col: color.RGBA{230, 70, 70, 255}},
39          {spd: 2.1, hp: 1, r: 6, col: color.RGBA{240, 150, 60, 255}},
40          {spd: 0.5, hp: 6, r: 15, col: color.RGBA{175, 85, 215, 255}},
41      }
42  )
```

**Line 37** declares `kinds` as a **slice** of `ent`. A slice (`[]T`) is Go's growable list. This one
holds three `ent` values used as **templates** — not as live enemies. Nothing on screen corresponds
to them; they are the blueprints a spawn is copied from.

- **Line 38** — the grunt: normal speed, 2 hit points, radius 9, red.
- **Line 39** — the sprinter: more than twice as fast, but 1 hit point and small (radius 6), orange.
- **Line 40** — the brute: very slow, 6 hit points, big (radius 15), purple.

Inside a `[]ent{...}` literal you may omit the type of each element, which is why each line is just
`{...}` and not `ent{...}`. Fields not mentioned (`x`, `y`, `dx`, `dy`) get Go's **zero value** —
`0` for numbers. Go has no uninitialised memory; every value starts zeroed.

**This is the most important Go idea in the file:** on line 69 a spawn does `e := kinds[i]`, and
that **copies** the struct. Go structs are values, not references. If you come from Python, Java or
JavaScript, your instinct will say `e` is a pointer to the template and that mutating `e.spd` on
line 71 would corrupt the blueprint for every future enemy. In Go it does not. `e` is an
independent copy the moment it is assigned.

---

## Lines 44–50 — the `ent` type

```go
44  // ent is a moving circle: player, bullet, enemy or health pack. Bullets carry a velocity in dx, dy;
45  // enemies keep a scalar spd and re-aim at the player every tick.
46  type ent struct {
47      x, y, dx, dy, spd, r float64
48      hp                   int
49      col                  color.RGBA
50  }
```

One struct for every object in the game. That is a deliberate trade: it keeps the file short, at the
cost of some fields being meaningless for some objects.

- **Line 47** — `x, y` is the centre position. `dx, dy` is a **velocity** — how far it moves each
  tick — and only bullets use it. `spd` is a **scalar speed** with no direction, and only enemies
  use it, because an enemy recomputes its direction every tick to chase a moving player. `r` is the
  radius, used for both drawing and collision.
- **Line 48** — `hp` is hit points. For enemies it's how many bullets they absorb; for the player
  it's health out of 100. A bullet's `hp` and a pack's `hp` are 0 and never read.
- **Line 49** — the colour it is drawn in. Because every entity carries its own `r` and `col`, the
  drawing code (line 204) never has to ask *what kind of thing is this* — a huge simplification.

Being honest about the cost: a health pack carries `hp`, `spd`, `dx` and `dy` that mean nothing. In
a bigger program you'd use separate types, or an interface. Here, one type keeps the whole game
readable in one screenful of struct.

---

## Lines 52–56 — the game state

```go
52  type game struct {
53      player                  ent
54      enemies, bullets, packs []ent
55      round, score, cooldown  int
56  }
```

Every piece of mutable state in the entire program lives in this one struct. There are no globals
that change. If you want to know what the game *is* at any instant, it is exactly these six fields.

- **Line 53** — the player: a single `ent`, stored by value, not a pointer.
- **Line 54** — three slices: the live enemies, the bullets in flight, and the uncollected health
  packs. Each starts as `nil` — in Go a `nil` slice has length 0 and `append` works on it fine, so
  there's nothing to initialise.
- **Line 55** — `round` counts from 1. `score` accumulates. `cooldown` counts down the ticks until
  the player may fire again.

---

## Lines 58–62 — starting a new game

```go
58  func newGame() *game {
59      g := &game{player: ent{x: screenW / 2, y: screenH / 2, r: playerRadius, hp: playerMaxHP, col: colPlayer}}
60      g.nextRound()
61      return g
62  }
```

**Line 58** — returns `*game`, a **pointer** to a game. Pointers are how Go lets several places
share and mutate one value. If this returned `game` (a value), every caller would get a private
copy and changes would go nowhere.

**Line 59** — `&game{...}` allocates a `game` and takes its address. The player is placed at the
centre of the screen (`640/2, 480/2`), at full health, with the radius and colour that make it draw
correctly. The three slices are not mentioned, so they're `nil` — an empty field with no enemies,
bullets or packs.

**Line 60** — immediately spawn round 1. Without this the game would open on an empty screen.

**Line 61** — hand the pointer back.

---

## Lines 64–75 — `nextRound`: spawning a wave

```go
64  // nextRound scales enemy count, speed and toughness with the round, and heals the survivor.
65  func (g *game) nextRound() {
66      g.round++
67      g.player.hp = min(g.player.hp+roundHeal, playerMaxHP)
68      for range g.round*2 + 2 {
69          e := kinds[rand.IntN(len(kinds))]
70          e.x, e.y = spawnEdge()
71          e.spd += float64(g.round) * 0.08
72          e.hp += g.round / 4
73          g.enemies = append(g.enemies, e)
74      }
75  }
```

**Line 65** — `func (g *game) nextRound()` is a **method**. The `(g *game)` part is the
**receiver**: it makes this a function you call *on* a game, as `g.nextRound()`. Because the
receiver is a pointer (`*game`), the method can modify the game. With a value receiver (`g game`)
it would modify a copy and nothing would happen.

**Line 66** — `g.round++` increments the round. So the first call takes it from 0 to 1.

**Line 67** — heal for surviving. `min(a, b)` is a **builtin** (since Go 1.21 — no import needed).
It caps the heal: `min(hp+15, 100)` can never exceed 100. Without the cap, a careful player would
climb past max health forever.

**Line 68** — `for range n` runs the loop body exactly `n` times (Go 1.22+). No index variable,
because we don't need one. The count is `round*2 + 2`: round 1 spawns 4 enemies, round 2 spawns 6,
round 10 spawns 22.

**Line 69** — pick a random archetype. `rand.IntN(k)` returns a random integer in `[0, k)` — zero up
to but *not including* `k`, which is exactly the valid index range of a slice of length `k`, so
`rand.IntN(len(kinds))` can never index out of bounds. As explained above, the assignment **copies**
the template.

**Line 70** — place it on a random screen edge (`spawnEdge`, line 78). Go lets a function return two
values and assign both at once.

**Line 71** — round scaling. Each round adds `0.08` to speed, on top of the archetype's base. A
sprinter (base 2.1) is doing 2.9 by round 10. `float64(g.round)` is an explicit **type conversion**:
`g.round` is an `int` and `e.spd` is a `float64`, and Go refuses to mix numeric types silently. That
strictness is a feature — it's where whole classes of bug come from in other languages.

**Line 72** — toughness scaling, using **integer division**. `g.round / 4` on `int`s discards the
remainder: rounds 1–3 add 0, rounds 4–7 add 1, rounds 8–10 add 2. So a round-10 grunt takes 4 hits
instead of 2.

**Line 73** — `append` adds to a slice and returns the (possibly reallocated) slice, which is why
you must assign the result back. `g.enemies = append(g.enemies, e)` is the idiom; forgetting the
assignment is the single most common Go slice mistake.

---

## Lines 77–83 — `spawnEdge`: where enemies come from

```go
77  // spawnEdge returns a random border point, so waves close in instead of spawning on the player.
78  func spawnEdge() (float64, float64) {
79      if rand.IntN(2) == 0 {
80          return float64(rand.IntN(screenW)), float64(rand.IntN(2) * screenH)
81      }
82      return float64(rand.IntN(2) * screenW), float64(rand.IntN(screenH))
83  }
```

**Line 78** — `(float64, float64)` declares two return values. Note there is no receiver: this is a
plain function, not a method, because it needs nothing from the game.

**Line 79** — flip a coin: `rand.IntN(2)` is 0 or 1.

**Line 80** — the heads case: a **horizontal** edge. `x` is anywhere across the width. `y` is
`rand.IntN(2) * screenH`, and that little trick is the point — `rand.IntN(2)` is 0 or 1, so `y` is
either `0` (top edge) or `480` (bottom edge). Never in between.

**Line 82** — the tails case, mirrored: `x` is `0` or `640` (left or right edge), `y` anywhere down
the height.

The result is a uniformly-picked point on the border of the screen. Enemies always walk *in* from
outside your field of fire. If they could spawn anywhere, one could appear on top of you and land a
free hit — unfair in a way the player can't see coming.

---

## Lines 85–95 — `filter`: the hardest function in the file

```go
85  // filter keeps the entities for which keep reports true, reusing the backing array so steady-state
86  // play doesn't allocate. keep takes a pointer, so it may mutate the entity it is judging.
87  func filter(es []ent, keep func(e *ent) bool) []ent {
88      live := es[:0]
89      for i := range es {
90          if keep(&es[i]) {
91              live = append(live, es[i])
92          }
93      }
94      return live
95  }
```

Take this one slowly. It is used three times (bullets, enemies, packs) and it is the reason the game
doesn't allocate memory while you play.

**Line 87** — the second parameter is a **function**: `keep func(e *ent) bool`. In Go, functions are
values you can pass around like any other. The caller supplies the *rule* for what survives; `filter`
supplies the *mechanism* for removing what doesn't.

**Line 88** — `live := es[:0]` is the trick. A Go slice is really three things: a pointer to an
array, a length, and a capacity. `es[:0]` makes a new slice that points at **the same array**, with
length 0 but the full capacity still available. So `live` and `es` share memory.

**Line 89** — `for i := range es` walks the indices `0, 1, 2, …`. We deliberately loop over the
index, not the value (`for _, e := range es`), because the next line needs the address of the real
element.

**Line 90** — `keep(&es[i])` passes a **pointer** to the element. Two consequences: nothing is
copied, and the callback can *modify* the entity it's inspecting. That's how a bullet moves itself
(line 154) inside what looks like a pure filter.

**Line 91** — the survivor is copied forward to the front of the shared array. This is safe only
because `live` can never overtake `i`: an element is written to index `len(live)`, and `len(live)`
is always `<= i`. You are overwriting slots you have already read.

**Line 94** — return the compacted slice. The caller assigns it back (`g.bullets = filter(...)`),
and the dead entries beyond the new length are simply forgotten.

**Why bother?** The naive version builds a fresh slice each tick, which allocates 60 times a second,
forever, and gives the garbage collector constant work. This version reuses one array for the life
of the program. This is a standard Go idiom, and it is worth learning — but it *is* an advanced one,
and if it doesn't click yet, treat `filter` as a black box: "keep the ones where the rule says true."

---

## Lines 97–100 — `done`: is the run over?

```go
 97  // done reports that the run has ended: the player is dead, or the final wave is clear.
 98  func (g *game) done() bool {
 99      return g.player.hp <= 0 || (g.round >= finalRound && len(g.enemies) == 0)
100  }
```

**Line 99** — two ways to end. Either health has hit zero (`hp <= 0` — not `== 0`, because a hit for
12 damage can take you from 5 to -7), **or** you are on the final round with no enemies left. The
parentheses are not required by Go's precedence rules, but they make the "this or that" shape
obvious at a glance.

---

## Lines 102–118 — `Update`: one tick of the simulation

```go
102  func (g *game) Update() error {
103      if g.done() {
104          if inpututil.IsKeyJustPressed(ebiten.KeyR) {
105              *g = *newGame()
106          }
107          return nil
108      }
109      g.movePlayer()
110      g.fire()
111      g.stepBullets()
112      g.stepEnemies()
113      g.stepPacks()
114      if len(g.enemies) == 0 && g.round < finalRound { // past finalRound a cleared wave is a win, not a new round
115          g.nextRound()
116      }
117      return nil
118  }
```

This is one of the three methods Ebitengine requires (with `Draw` and `Layout`). Together they
satisfy the `ebiten.Game` **interface** — Go's way of saying "any type with these methods can be
used here." We never mention `ebiten.Game` by name; having the methods is enough.

**Line 103** — if the run is over, the simulation is frozen. Nothing moves, nothing spawns.

**Line 104** — `IsKeyJustPressed` is true **only on the tick the key goes down**, unlike
`IsKeyPressed` which is true the whole time it's held. Here it prevents one long press of R from
restarting the game 60 times a second.

**Line 105** — `*g = *newGame()` deserves a close look. It reads: "take the game `newGame()` built,
copy its whole contents *into the memory `g` points at*." We cannot simply write `g = newGame()`,
because that would only repoint our local variable — Ebitengine is holding the *original* pointer
(handed to it on line 240) and would carry on running the old game. Overwriting through the pointer
resets the state everyone shares.

**Line 107** — `return nil` means "no error, keep running." Returning a non-nil error is how a game
tells Ebitengine to stop.

**Lines 109–113** — one tick, in order. The order is load-bearing:

1. `movePlayer` — read the keyboard, move the player.
2. `fire` — read the mouse, maybe spawn a bullet.
3. `stepBullets` — move bullets, hit enemies (an enemy's `hp` drops here).
4. `stepEnemies` — remove enemies whose `hp` just dropped to 0, move the rest, apply contact damage.
5. `stepPacks` — pick up any health pack the player is standing on.

Because bullets resolve *before* enemies, an enemy killed on line 160 is removed on line 172 in the
**same tick**. If the order were flipped, every kill would take an extra tick to register.

**Line 114** — the wave is clear, so start the next one — **but only below `finalRound`**. That
second condition is the whole win condition. Without it, clearing round 10 would immediately spawn
round 11 in the same tick, `done()` would never once be true, and the "YOU SURVIVED" screen could
never appear. (That was a real bug in this file. One missing condition made an entire feature
unreachable, while the game still looked like it worked.)

---

## Lines 120–127 — `held`: reading a direction key

```go
120  func held(keys ...ebiten.Key) float64 {
121      for _, k := range keys {
122          if ebiten.IsKeyPressed(k) {
123              return 1
124          }
125      }
126      return 0
127  }
```

**Line 120** — `keys ...ebiten.Key` is **variadic**: call it with any number of keys, and inside the
function `keys` is a slice. That's what lets line 130 treat `D` and `→` as the same input.

**Lines 121–124** — if *any* of the given keys is down, return `1`.

**Line 126** — otherwise `0`.

Returning a `float64` of 1 or 0 (rather than a `bool`) looks odd until you see the caller: it turns
"is a key down" into arithmetic. See line 130.

---

## Lines 129–137 — `movePlayer`

```go
129  func (g *game) movePlayer() {
130      dx := held(ebiten.KeyD, ebiten.KeyArrowRight) - held(ebiten.KeyA, ebiten.KeyArrowLeft)
131      dy := held(ebiten.KeyS, ebiten.KeyArrowDown) - held(ebiten.KeyW, ebiten.KeyArrowUp)
132      if dx != 0 && dy != 0 { // keep diagonal speed equal to cardinal speed
133          dx, dy = dx/math.Sqrt2, dy/math.Sqrt2
134      }
135      g.player.x = min(max(g.player.x+dx*playerSpeed, playerRadius), screenW-playerRadius)
136      g.player.y = min(max(g.player.y+dy*playerSpeed, playerRadius), screenH-playerRadius)
137  }
```

**Line 130** — *right minus left*. Holding D gives `1 - 0 = 1`. Holding A gives `0 - 1 = -1`.
Holding both gives `1 - 1 = 0`, which is exactly right: opposite keys should cancel. One subtraction
replaces four branches.

**Line 131** — the same for the vertical axis. Note `S` (down) is **positive**: on a screen, `y`
grows *downward*. Top-left is `(0, 0)`. This trips up everyone once.

**Lines 132–134** — the diagonal fix. Moving right *and* down gives a vector of length
`√(1² + 1²) = √2 ≈ 1.41`, so a naive diagonal would be 41% faster than moving straight — a classic
bug in beginner games. Dividing both components by `√2` brings the length back to 1.
`math.Sqrt2` is a precomputed constant; `if dx != 0 && dy != 0` ensures we only do this when
actually moving diagonally.

**Lines 135–136** — move, then clamp to the screen. Read it inside-out:
`g.player.x + dx*playerSpeed` is the new position; `max(..., playerRadius)` stops the circle's edge
crossing the left wall; `min(..., screenW-playerRadius)` stops it crossing the right. The radius
appears because `x` is the *centre* — clamping to `0` would let half the player hang off-screen.

---

## Lines 139–149 — `fire`: shooting

```go
139  func (g *game) fire() {
140      g.cooldown = max(g.cooldown-1, 0)
141      if g.cooldown > 0 || !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
142          return
143      }
144      mx, my := ebiten.CursorPosition()
145      a := math.Atan2(float64(my)-g.player.y, float64(mx)-g.player.x) // Atan2 stays defined with the cursor on the player
146      g.bullets = append(g.bullets, ent{x: g.player.x, y: g.player.y, r: bulletRadius, col: colBullet,
147          dx: math.Cos(a) * bulletSpeed, dy: math.Sin(a) * bulletSpeed})
148      g.cooldown = fireCooldown
149  }
```

**Line 140** — tick the cooldown down by one, but never below zero. `max(x-1, 0)` in one line, with
no `if`.

**Line 141** — leave early if we're still cooling down, **or** the left mouse button isn't held. Go
evaluates `||` left to right and **short-circuits**, so when the cooldown is still running it never
even asks about the mouse.

**Line 144** — where is the cursor? Returns `int` pixels.

**Line 145** — the aim angle. `math.Atan2(y, x)` — mind the argument order, `y` first — gives the
angle of the vector from the player to the cursor, in radians, correct in all four quadrants. The
comment records the edge case: with the cursor exactly on the player both arguments are 0, and
plain division would be `0/0` (NaN), but `Atan2(0, 0)` is defined and simply returns 0. The bullet
flies right. No crash, no NaN poisoning the position.

**Lines 146–147** — build the bullet. It starts at the player's centre, and its velocity is the aim
angle turned back into components: `cos(a)` and `sin(a)` give a vector of length exactly 1
(a unit vector), and multiplying by `bulletSpeed` scales it to 8 pixels per tick. `spd` and `hp` are
left at zero — a bullet has no use for them.

**Line 148** — reset the cooldown, so the next shot is 8 ticks away.

---

## Lines 151–166 — `stepBullets`

```go
151  // stepBullets advances bullets, drops off-screen ones, and spends a bullet on the first enemy hit.
152  func (g *game) stepBullets() {
153      g.bullets = filter(g.bullets, func(b *ent) bool {
154          b.x, b.y = b.x+b.dx, b.y+b.dy
155          if b.x < 0 || b.x > screenW || b.y < 0 || b.y > screenH {
156              return false
157          }
158          for i := range g.enemies {
159              if e := &g.enemies[i]; math.Hypot(e.x-b.x, e.y-b.y) < e.r+b.r {
160                  e.hp--
161                  return false
162              }
163          }
164          return true
165      })
166  }
```

**Line 153** — call `filter`, passing an **anonymous function** (a function with no name, written
inline). It is also a **closure**: it uses `g` from the enclosing method, so it can reach the enemy
list even though `filter` knows nothing about games. Returning `true` keeps the bullet; `false`
deletes it.

**Line 154** — move the bullet by its velocity. This *mutates the bullet* through the pointer — the
"filter" is quietly also the "update," which is the one liberty this design takes.

**Lines 155–157** — off the screen? Drop it. Without this, every missed shot would fly forever and
the slice would grow without bound.

**Line 158** — check the bullet against every live enemy. Brute force, no spatial index. At a few
hundred entities that is genuinely faster than anything cleverer, because the cost of *maintaining*
a spatial structure exceeds the cost of the comparisons.

**Line 159** — two things in one line. `e := &g.enemies[i];` takes a pointer to the enemy (so
`e.hp--` on the next line hits the real one, not a copy) — Go's `if` allows a statement before the
condition, separated by a semicolon. Then the collision test: `math.Hypot(dx, dy)` is
`√(dx² + dy²)`, the distance between the two centres. Two circles overlap exactly when that distance
is less than the sum of their radii. That is the entire collision system.

**Lines 160–161** — a hit: the enemy loses one hit point, and the bullet is consumed (`return
false`). Note the enemy is **not** removed here even if its `hp` reaches 0 — removal is
`stepEnemies`' job, on the very next line of `Update`. Returning immediately means one bullet can
never hit two enemies.

**Line 164** — no wall, no enemy: the bullet survives to the next tick.

---

## Lines 168–188 — `stepEnemies`

```go
168  // stepEnemies homes enemies on the player, banks score and rolls a pack drop for the dead, and
169  // trades an enemy for player HP on contact.
170  func (g *game) stepEnemies() {
171      g.enemies = filter(g.enemies, func(e *ent) bool {
172          if e.hp <= 0 {
173              g.score += 10 * g.round
174              if rand.IntN(packChance) == 0 {
175                  g.packs = append(g.packs, ent{x: e.x, y: e.y, r: packRadius, col: colPack})
176              }
177              return false
178          }
179          dx, dy := g.player.x-e.x, g.player.y-e.y
180          d := math.Hypot(dx, dy)
181          if d < g.player.r+e.r { // guards the division below: d is never 0 past this point
182              g.player.hp -= contactDamage
183              return false
184          }
185          e.x, e.y = e.x+dx/d*e.spd, e.y+dy/d*e.spd
186          return true
187      })
188  }
```

**Line 172** — dead? (Its `hp` was decremented by a bullet moments ago, in `stepBullets`.)

**Line 173** — score. `10 * g.round` means a kill on round 10 is worth ten times a kill on round 1 —
so a long run's score is dominated by its late waves, and dying on round 9 costs you a lot.

**Lines 174–176** — roll for a health pack. `rand.IntN(4)` gives 0, 1, 2 or 3 with equal chance, so
`== 0` is a clean 25%. The pack is dropped **where the enemy died** (`x: e.x, y: e.y`), which is the
design in miniature: the reward appears in the middle of where the fight was, so collecting it means
walking back into contested space instead of kiting in a safe circle forever.

**Line 177** — the corpse leaves the slice.

**Line 179** — the vector *from* the enemy *to* the player. (Player minus enemy — get this backwards
and your enemies flee.)

**Line 180** — its length: the distance to the player.

**Lines 181–183** — contact. If the circles overlap, the player takes 12 damage and the enemy is
consumed — it doesn't linger and grind the player down; it's a one-time trade. This is also why the
player can be reduced below zero health, which is why `done()` tests `hp <= 0`.

**Line 185** — the chase. `dx/d` and `dy/d` are the direction to the player as a unit vector
(dividing a vector by its own length always yields length 1), and multiplying by `e.spd` steps that
far along it. This is why enemies track a moving target: the direction is recomputed from scratch
every single tick.

And now the subtle part, which the comment on line 181 exists to record: this line **divides by
`d`**, so `d` must never be zero. It cannot be — if `d` were 0 the enemy would be exactly on the
player, which means `d < g.player.r + e.r` was true, and line 183 already returned. The early return
is not just gameplay; it is what makes the division safe. Delete the contact check and you introduce
a divide-by-zero that produces `NaN` coordinates and silently corrupts the enemy forever.

---

## Lines 190–200 — `stepPacks`

```go
190  // stepPacks collects any health pack the player walks over.
191  func (g *game) stepPacks() {
192      g.packs = filter(g.packs, func(p *ent) bool {
193          if math.Hypot(g.player.x-p.x, g.player.y-p.y) > g.player.r+p.r {
194              return true
195          }
196          g.player.hp = min(g.player.hp+packHeal, playerMaxHP)
197          g.score += packScore
198          return false
199      })
200  }
```

**Line 193** — the same circle-overlap test, inverted: if the player is *further* than the two radii
combined, there's no contact, so keep the pack lying there (`return true`). Packs never expire.

**Line 196** — heal, capped at 100 by `min` — same guard as line 67.

**Line 197** — a small score bonus, so a pack is worth grabbing even at full health.

**Line 198** — consumed; remove it.

---

## Lines 202–206 — `circle`: the only drawing primitive

```go
202  // circle paints one entity. Every drawn thing is a filled circle, so radius and colour ride on the
203  // ent itself rather than being looked up per kind at draw time.
204  func circle(dst *ebiten.Image, e ent) {
205      vector.FillCircle(dst, float32(e.x), float32(e.y), float32(e.r), e.col, true)
206  }
```

**Line 204** — takes the image to paint on and the entity to paint. The `ent` is passed **by value**
— a copy — which is fine and even good here: drawing must not modify the game, and a value parameter
makes that impossible by construction.

**Line 205** — Ebitengine's shape drawing wants `float32`, our game computes in `float64`, so each
coordinate is converted explicitly. The final `true` turns on antialiasing, which is what keeps the
circles from looking jagged. Because `r` and `col` come from the entity, this one function draws the
player, an enemy, a bullet and a health pack without knowing which is which.

---

## Lines 208–225 — `Draw`: painting the frame

```go
208  func (g *game) Draw(screen *ebiten.Image) {
209      screen.Fill(colBG)
210      for _, es := range [][]ent{g.packs, g.enemies, g.bullets} { // packs first: they sit under the fight
211          for _, e := range es {
212              circle(screen, e)
213          }
214      }
215      if g.player.hp > 0 { // a dead player leaves the field, so the end screen isn't drawn over a corpse
216          circle(screen, g.player)
217      }
```

**Line 209** — repaint the background over last frame's image. Skip this and everything smears.

**Line 210** — `[][]ent{g.packs, g.enemies, g.bullets}` is a slice *of slices*, built inline purely
so one loop can walk all three lists. The order **is** the draw order, because painting is
back-to-front, like stacking paper: packs are painted first and so appear *underneath* the enemies
fighting over them; bullets are painted last of the three and read clearly against everything.

**Lines 215–216** — the player goes on top, and only if alive. It's drawn separately from the loop
for exactly that reason — it's the one entity with a condition attached.

```go
218      label(screen, fmt.Sprintf("ROUND %d/%d   HP %d   SCORE %d   LEFT %d", g.round, finalRound, max(g.player.hp, 0), g.score, len(g.enemies)), 8, 8)
219      switch {
220      case g.player.hp <= 0:
221          label(screen, fmt.Sprintf("GAME OVER\nfell on round %d of %d  ·  %d points\npress R to try again", g.round, finalRound, g.score), screenW/2-115, screenH/2-24)
222      case g.done():
223          label(screen, fmt.Sprintf("YOU SURVIVED\nall %d rounds cleared  ·  %d points  ·  %d HP left\npress R to play again", finalRound, g.score, g.player.hp), screenW/2-155, screenH/2-24)
224      }
225  }
```

**Line 218** — the HUD, at the top-left corner (8, 8). `fmt.Sprintf` builds a string by substituting
values into a template: `%d` means "an integer goes here." Note `max(g.player.hp, 0)` — health can
genuinely be negative in the state, but showing "HP -7" would look broken, so the display floors it
at 0. This is a `Draw`-only cosmetic decision, and it belongs here rather than in `Update`, which
must keep the true value.

**Line 219** — a `switch` with no subject: each `case` is just a boolean, and the **first** true one
wins. That ordering matters. If you are dead on the final round, both `hp <= 0` and `done()` are
true, and putting death first guarantees a death shows "GAME OVER" and never "YOU SURVIVED."

**Line 221** — the loss screen. `\n` is a newline; `label` (line 230) sets the line spacing.

**Line 223** — the win screen — the one that was unreachable until line 114's condition was fixed.

The magic numbers `-115` and `-155` are rough half-widths of the text, centring it by eye. In a
larger game you'd measure the string; here, two hand-tuned numbers are cheaper and just as good.

---

## Lines 227–233 — `label`: drawing text

```go
227  func label(dst *ebiten.Image, s string, x, y float64) {
228      op := &text.DrawOptions{}
229      op.GeoM.Translate(x, y)
230      op.LineSpacing = 16
231      op.ColorScale.ScaleWithColor(color.White)
232      text.Draw(dst, s, face, op)
233  }
```

**Line 228** — an options struct, created empty and then filled in. This is a common Go pattern when
a call has many optional settings — instead of a function with eight parameters, you get one options
value you configure.

**Line 229** — `GeoM` is a geometry matrix: the transform (move, rotate, scale) applied to the text.
`Translate(x, y)` moves it to the requested position. Without this, everything would draw at (0, 0).

**Line 230** — how far apart stacked lines sit, in pixels. The font is 13 pixels tall, so 16 leaves
a small gap. Only the multi-line end screens notice.

**Line 231** — paint the text white. `ColorScale` *multiplies* the source colour, so this is
tinting, not overwriting.

**Line 232** — draw string `s` in `face` (built once, back on line 29) with those options.

---

## Line 235 — `Layout`

```go
235  func (g *game) Layout(int, int) (int, int) { return screenW, screenH }
```

The third method Ebitengine requires. It answers: "what is the game's internal resolution?" The two
`int` parameters are the actual window size in pixels — and we **ignore** them, which is why they
have no names (Go permits unnamed parameters when you don't use them).

By always answering 640×480, the game logic lives in a fixed coordinate system no matter how the
window is resized; Ebitengine scales the result to fit. That is why nothing anywhere else in the
file ever has to think about window size.

---

## Lines 237–243 — `main`: the entry point

```go
237  func main() {
238      ebiten.SetWindowSize(screenW, screenH)
239      ebiten.SetWindowTitle("Survival")
240      if err := ebiten.RunGame(newGame()); err != nil {
241          log.Fatal(err)
242      }
243  }
```

**Line 237** — where the program starts.

**Lines 238–239** — open a 640×480 window titled "Survival".

**Line 240** — hand a fresh game to Ebitengine and start the loop. `RunGame` **blocks**: it does not
return until the window is closed or something fails. From this moment on, the engine drives — it
calls `Update` 60 times a second and `Draw` once per frame, forever. Our code never calls them.

`if err := ...; err != nil` is *the* Go error-handling idiom: run the call, capture the error into a
variable scoped to the `if`, and check it immediately. Go has no exceptions; errors are ordinary
values you are expected to look at.

**Line 241** — `log.Fatal` prints the error and exits with a non-zero status. Reached only if
Ebitengine itself fails — no display, no GPU.

---

## The Go features you just met

| Feature | First seen | What to remember |
| --- | --- | --- |
| `package main` + `func main()` | 2, 237 | A runnable program, and where it starts |
| Slices `[]T` | 37 | Growable list; `append` **returns** the new slice — assign it back |
| Structs are **values** | 69 | Assigning a struct copies it. It is not a reference |
| Pointers `*T` / `&x` | 58, 90 | Needed whenever a function must *modify* its argument |
| Methods and receivers | 65 | `func (g *game) f()` — a pointer receiver can mutate; a value one cannot |
| Zero values | 38 | Nothing is uninitialised; unset numbers are `0`, unset slices are `nil` |
| Multiple returns | 78 | `return x, y` — no tuples, no out-params |
| Variadic `...T` | 120 | Any number of arguments, seen inside as a slice |
| Closures | 153 | An inline function that captures surrounding variables (here, `g`) |
| `min` / `max` builtins | 67 | No import; clamps in one expression |
| `for range n` | 68 | Repeat `n` times, no index |
| Explicit conversions | 71 | `float64(i)` — Go never mixes numeric types silently |
| Interfaces, implicitly | 102 | `Update`/`Draw`/`Layout` satisfy `ebiten.Game`; you never say so |
| `if err != nil` | 240 | Errors are values, checked on the spot |

## Where to poke it

Small changes with visible results, in rough order of difficulty:

1. Make bullets faster: `bulletSpeed` (line 21). Try 16.
2. Make the game brutal: `contactDamage` (line 22) to 30.
3. Make packs common: `packChance` (line 24) to 2 — a 50% drop.
4. Add a fourth enemy archetype: one line in `kinds` (lines 37–41). No other change is needed;
   spawning, drawing and collision all pick it up for free. That's the payoff of the one-`ent`
   design — see if you can say exactly *why* no other code has to change.
5. Make bullets pierce: in `stepBullets`, `return false` on line 161 spends the bullet on a hit.
   What happens if it keeps going? (Careful: what stops it hitting the same enemy 60 times?)
