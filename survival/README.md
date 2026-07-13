# Survival

A round-based survival shooter in a single Go file. Waves of enemies close in from the screen
edges; clear a wave and the next round spawns more of them, faster and tougher. Contact costs
health. Reach zero and the run is over — **clear all ten rounds and you win**, with your score and
remaining health on the end screen.

![status](https://img.shields.io/badge/go-1.26-blue) ![deps](https://img.shields.io/badge/direct%20deps-2-green)

New to the code? [`WALKTHROUGH.md`](WALKTHROUGH.md) explains `main.go` line by line.

## Play

```sh
go run .
```

| Input | Action |
| --- | --- |
| `W` `A` `S` `D` / arrows | Move |
| Mouse | Aim |
| Left click (hold) | Fire |
| `R` | Restart after a win or a loss |

The HUD shows the round (`3/10`), health, score and how many enemies are left in the wave.
Score per kill scales with the round, so the last waves are worth far more than the first ones.

### Enemies

Three archetypes spawn at random, and every one of them gets faster and tougher as the rounds go up:

| Enemy | Colour | Behaviour |
| --- | --- | --- |
| Grunt | red | Steady speed, 2 hit points. The baseline threat. |
| Sprinter | orange | Fast and small, dies to a single bullet. Punishes a slow reaction. |
| Brute | purple | Slow, large, 6 hit points. Soaks a whole magazine while the rest close in. |

### Health packs

One kill in four drops a blue health pack where the enemy died. Walking over it heals 20 HP (capped
at 100) and adds 25 to the score. Packs never expire, so a cleared corner of the map is worth
returning to — which is the point: it pulls you out of a safe kiting circle.

## Build

```sh
go build -o survival .   # produces a standalone binary
./survival
```

Requires Go 1.26+. On Linux the binary needs `libX11` and `libGL` at runtime — present on any
normal desktop install. No cgo, no C toolchain.

## Design

Everything lives in `main.go`. The game is an [Ebitengine](https://ebitengine.org) `Game`:
`Update` runs the simulation at a fixed 60 ticks per second, `Draw` paints the frame.

- **One entity type.** The player, bullets, enemies and health packs are all the same `ent` struct —
  a circle with a position, a radius and a colour. Bullets store a fixed velocity in `dx, dy`;
  enemies store a scalar `spd` and re-aim at the player every tick. Because every drawn thing is a
  circle carrying its own radius and colour, `Draw` paints all four through one `circle` call.
- **Enemy archetypes as prototypes.** `kinds` is a slice of `ent` values used as templates. A spawn
  copies one (`e := kinds[i]` — Go structs are values, so this is a real copy), then scales its
  speed and hit points by the round.
- **Circle collision.** Distance between centres versus the sum of radii. No spatial index: at a few
  hundred entities the brute-force pass is far cheaper than maintaining one.
- **In-place filtering.** One `filter` helper compacts bullets, enemies and packs with `s[:0]` reuse,
  so steady-state play does not allocate.

## Tuning

The `const` block at the top of `main.go` holds every gameplay knob: player speed and health, bullet
speed, fire cooldown, contact damage, per-round heal, the health-pack numbers (`packHeal`,
`packScore`, `packChance`), and `finalRound` — the wave that ends the run. Enemy count
(`round*2+2`), speed and hit points scale in `nextRound`; per-archetype base stats live in the
`kinds` table.

## License

MIT
