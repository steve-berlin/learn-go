# Survival

A round-based survival shooter in a single ~200-line Go file. Waves of enemies close in from
the screen edges; clear a wave and the next round spawns more of them, faster and tougher.
Contact costs health, and the run ends when health hits zero.

![status](https://img.shields.io/badge/go-1.26-blue) ![deps](https://img.shields.io/badge/direct%20deps-2-green)

## Play

```sh
go run .
```

| Input | Action |
| --- | --- |
| `W` `A` `S` `D` / arrows | Move |
| Mouse | Aim |
| Left click (hold) | Fire |
| `R` | Restart after game over |

The HUD shows the current round, health, score and how many enemies are left in the wave.
Score per kill scales with the round, so surviving deeper is worth far more than farming early waves.

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

- **One entity type.** Player, bullets and enemies are all the same `ent` struct — a circle with a
  position. Bullets store a fixed velocity; enemies store a speed and re-aim at the player each tick.
- **Circle collision.** Distance between centres versus the sum of radii. No spatial index: at a few
  hundred entities the brute-force pass is far cheaper than maintaining one.
- **In-place filtering.** Dead bullets and enemies are compacted with `s[:0]` reuse, so steady-state
  play does not allocate.

## Tuning

The `const` block at the top of `main.go` holds every gameplay knob: player speed and health,
bullet speed, fire cooldown, contact damage, per-round heal. Enemy count (`round*2+2`), speed and
hit points scale in `nextRound`.

## License

MIT
