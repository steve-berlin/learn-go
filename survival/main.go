// Command survival is a round-based survival shooter: WASD to move, mouse to aim, click to fire.
package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

const (
	screenW, screenH                        = 640, 480
	playerRadius, playerSpeed, playerMaxHP  = 10, 3, 100
	bulletRadius, bulletSpeed, fireCooldown = 3, 8, 8 // cooldown in ticks
	contactDamage, roundHeal                = 12, 15
	packRadius, packHeal, packScore         = 6, 20, 25
	packChance                              = 4  // one kill in packChance drops a health pack
	finalRound                              = 10 // clear this wave and the run is won
)

var (
	face      = text.NewGoXFace(basicfont.Face7x13)
	colBG     = color.RGBA{18, 18, 24, 255}
	colPlayer = color.RGBA{80, 220, 120, 255}
	colBullet = color.RGBA{250, 210, 90, 255}
	colPack   = color.RGBA{110, 200, 255, 255}

	// kinds are the enemy archetypes — steady grunt, fragile sprinter, slow brute — drawn at random
	// per spawn and then scaled by the round in nextRound.
	kinds = []ent{
		{spd: 1.0, hp: 2, r: 9, col: color.RGBA{230, 70, 70, 255}},
		{spd: 2.1, hp: 1, r: 6, col: color.RGBA{240, 150, 60, 255}},
		{spd: 0.5, hp: 6, r: 15, col: color.RGBA{175, 85, 215, 255}},
	}
)

// ent is a moving circle: player, bullet, enemy or health pack. Bullets carry a velocity in dx, dy;
// enemies keep a scalar spd and re-aim at the player every tick.
type ent struct {
	x, y, dx, dy, spd, r float64
	hp                   int
	col                  color.RGBA
}

type game struct {
	player                  ent
	enemies, bullets, packs []ent
	round, score, cooldown  int
}

func newGame() *game {
	g := &game{player: ent{x: screenW / 2, y: screenH / 2, r: playerRadius, hp: playerMaxHP, col: colPlayer}}
	g.nextRound()
	return g
}

// nextRound scales enemy count, speed and toughness with the round, and heals the survivor.
func (g *game) nextRound() {
	g.round++
	g.player.hp = min(g.player.hp+roundHeal, playerMaxHP)
	for range g.round*2 + 2 {
		e := kinds[rand.IntN(len(kinds))]
		e.x, e.y = spawnEdge()
		e.spd += float64(g.round) * 0.08
		e.hp += g.round / 4
		g.enemies = append(g.enemies, e)
	}
}

// spawnEdge returns a random border point, so waves close in instead of spawning on the player.
func spawnEdge() (float64, float64) {
	if rand.IntN(2) == 0 {
		return float64(rand.IntN(screenW)), float64(rand.IntN(2) * screenH)
	}
	return float64(rand.IntN(2) * screenW), float64(rand.IntN(screenH))
}

// filter keeps the entities for which keep reports true, reusing the backing array so steady-state
// play doesn't allocate. keep takes a pointer, so it may mutate the entity it is judging.
func filter(es []ent, keep func(e *ent) bool) []ent {
	live := es[:0]
	for i := range es {
		if keep(&es[i]) {
			live = append(live, es[i])
		}
	}
	return live
}

// done reports that the run has ended: the player is dead, or the final wave is clear.
func (g *game) done() bool {
	return g.player.hp <= 0 || (g.round >= finalRound && len(g.enemies) == 0)
}

func (g *game) Update() error {
	if g.done() {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			*g = *newGame()
		}
		return nil
	}
	g.movePlayer()
	g.fire()
	g.stepBullets()
	g.stepEnemies()
	g.stepPacks()
	if len(g.enemies) == 0 && g.round < finalRound { // past finalRound a cleared wave is a win, not a new round
		g.nextRound()
	}
	return nil
}

func held(keys ...ebiten.Key) float64 {
	for _, k := range keys {
		if ebiten.IsKeyPressed(k) {
			return 1
		}
	}
	return 0
}

func (g *game) movePlayer() {
	dx := held(ebiten.KeyD, ebiten.KeyArrowRight) - held(ebiten.KeyA, ebiten.KeyArrowLeft)
	dy := held(ebiten.KeyS, ebiten.KeyArrowDown) - held(ebiten.KeyW, ebiten.KeyArrowUp)
	if dx != 0 && dy != 0 { // keep diagonal speed equal to cardinal speed
		dx, dy = dx/math.Sqrt2, dy/math.Sqrt2
	}
	g.player.x = min(max(g.player.x+dx*playerSpeed, playerRadius), screenW-playerRadius)
	g.player.y = min(max(g.player.y+dy*playerSpeed, playerRadius), screenH-playerRadius)
}

func (g *game) fire() {
	g.cooldown = max(g.cooldown-1, 0)
	if g.cooldown > 0 || !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		return
	}
	mx, my := ebiten.CursorPosition()
	a := math.Atan2(float64(my)-g.player.y, float64(mx)-g.player.x) // Atan2 stays defined with the cursor on the player
	g.bullets = append(g.bullets, ent{x: g.player.x, y: g.player.y, r: bulletRadius, col: colBullet,
		dx: math.Cos(a) * bulletSpeed, dy: math.Sin(a) * bulletSpeed})
	g.cooldown = fireCooldown
}

// stepBullets advances bullets, drops off-screen ones, and spends a bullet on the first enemy hit.
func (g *game) stepBullets() {
	g.bullets = filter(g.bullets, func(b *ent) bool {
		b.x, b.y = b.x+b.dx, b.y+b.dy
		if b.x < 0 || b.x > screenW || b.y < 0 || b.y > screenH {
			return false
		}
		for i := range g.enemies {
			if e := &g.enemies[i]; math.Hypot(e.x-b.x, e.y-b.y) < e.r+b.r {
				e.hp--
				return false
			}
		}
		return true
	})
}

// stepEnemies homes enemies on the player, banks score and rolls a pack drop for the dead, and
// trades an enemy for player HP on contact.
func (g *game) stepEnemies() {
	g.enemies = filter(g.enemies, func(e *ent) bool {
		if e.hp <= 0 {
			g.score += 10 * g.round
			if rand.IntN(packChance) == 0 {
				g.packs = append(g.packs, ent{x: e.x, y: e.y, r: packRadius, col: colPack})
			}
			return false
		}
		dx, dy := g.player.x-e.x, g.player.y-e.y
		d := math.Hypot(dx, dy)
		if d < g.player.r+e.r { // guards the division below: d is never 0 past this point
			g.player.hp -= contactDamage
			return false
		}
		e.x, e.y = e.x+dx/d*e.spd, e.y+dy/d*e.spd
		return true
	})
}

// stepPacks collects any health pack the player walks over.
func (g *game) stepPacks() {
	g.packs = filter(g.packs, func(p *ent) bool {
		if math.Hypot(g.player.x-p.x, g.player.y-p.y) > g.player.r+p.r {
			return true
		}
		g.player.hp = min(g.player.hp+packHeal, playerMaxHP)
		g.score += packScore
		return false
	})
}

// circle paints one entity. Every drawn thing is a filled circle, so radius and colour ride on the
// ent itself rather than being looked up per kind at draw time.
func circle(dst *ebiten.Image, e ent) {
	vector.FillCircle(dst, float32(e.x), float32(e.y), float32(e.r), e.col, true)
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	for _, es := range [][]ent{g.packs, g.enemies, g.bullets} { // packs first: they sit under the fight
		for _, e := range es {
			circle(screen, e)
		}
	}
	if g.player.hp > 0 { // a dead player leaves the field, so the end screen isn't drawn over a corpse
		circle(screen, g.player)
	}
	label(screen, fmt.Sprintf("ROUND %d/%d   HP %d   SCORE %d   LEFT %d", g.round, finalRound, max(g.player.hp, 0), g.score, len(g.enemies)), 8, 8)
	switch {
	case g.player.hp <= 0:
		label(screen, fmt.Sprintf("GAME OVER\nfell on round %d of %d  ·  %d points\npress R to try again", g.round, finalRound, g.score), screenW/2-115, screenH/2-24)
	case g.done():
		label(screen, fmt.Sprintf("YOU SURVIVED\nall %d rounds cleared  ·  %d points  ·  %d HP left\npress R to play again", finalRound, g.score, g.player.hp), screenW/2-155, screenH/2-24)
	}
}

func label(dst *ebiten.Image, s string, x, y float64) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.LineSpacing = 16
	op.ColorScale.ScaleWithColor(color.White)
	text.Draw(dst, s, face, op)
}

func (g *game) Layout(int, int) (int, int) { return screenW, screenH }

func main() {
	ebiten.SetWindowSize(screenW, screenH)
	ebiten.SetWindowTitle("Survival")
	if err := ebiten.RunGame(newGame()); err != nil {
		log.Fatal(err)
	}
}
