# Survival — project instructions

Round-based survival shooter. Go + [Ebitengine](https://ebitengine.org). See `README.md` for
gameplay, controls and design notes; this file is the contract for working on the code.

Own module (`github.com/steve-berlin/learn-go/survival`) inside the multi-project `learn-go` repo.
Run every command from this directory, not the repo root.

## Hard constraints

- **`main.go` is the whole program and must stay under 200 lines.** Currently 198. Any new feature
  has to pay for itself — find lines elsewhere or make the case for the trade before writing it.
- **Minimal dependencies.** Two direct: `ebiten/v2` (window, GPU, input) and `x/image` (bitmap font).
  Adding a third needs justification; the stdlib usually already has it.
- **No cgo.** Ebitengine v2.6+ builds pure-Go on Linux. Keep it that way — it's why the binary is
  portable and the build is fast.
- Single file, single package `main`. No `internal/`, no packages, no config files.

## Conventions

- Go 1.26. Use current stdlib: `math/rand/v2`, builtin `min`/`max`, range-over-int (`for range n`).
- Gameplay constants live in the `const` block at the top of `main.go` — never inline a magic number
  in the simulation.
- `Update` mutates state, `Draw` only paints. Nothing in `Draw` may change the game.
- Filter slices in place (`live := s[:0]`) so steady-state play doesn't allocate.
- Comment the *why* (a non-obvious invariant, a numerical edge case), never the *what*.

## Before every commit

```sh
gofmt -l .          # must print nothing
go vet ./...        # must be clean
go build ./...      # must succeed
wc -l main.go       # must be < 200
```

The game is interactive and needs a display; **Steve runs and plays it**, agents do not.
A 4-second `timeout 4 ./survival` is fine as a crash smoke test, nothing more.

## Git

Commit with Conventional Commits (`feat:`, `fix:`, `perf:`, `docs:`, `refactor:`).
Work is pushed to the remote after Steve confirms the change.
