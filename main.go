// Command survival is a round-based survival shooter: move with WASD, aim with
// the mouse, hold left-click to fire. Clearing a wave starts the next, harder round.
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
	enemyRadius, contactDamage, roundHeal   = 9, 12, 15
)

var (
	face      = text.NewGoXFace(basicfont.Face7x13)
	colBG     = color.RGBA{18, 18, 24, 255}
	colAim    = color.RGBA{70, 70, 90, 255}
	colPlayer = color.RGBA{80, 220, 120, 255}
	colEnemy  = color.RGBA{230, 70, 70, 255}
	colBullet = color.RGBA{250, 210, 90, 255}
)

// ent is a moving circle: bullets carry a velocity in dx, dy; enemies keep a scalar spd.
type ent struct {
	x, y, dx, dy, spd float64
	hp                int
}

type game struct {
	player                 ent
	enemies, bullets       []ent
	round, score, cooldown int
}

func newGame() *game {
	g := &game{player: ent{x: screenW / 2, y: screenH / 2, hp: playerMaxHP}}
	g.nextRound()
	return g
}

// nextRound scales count, speed and toughness with the round, and heals the survivor.
func (g *game) nextRound() {
	g.round++
	g.player.hp = min(g.player.hp+roundHeal, playerMaxHP)
	for range g.round*2 + 2 {
		x, y := spawnEdge()
		g.enemies = append(g.enemies, ent{x: x, y: y, spd: 0.7 + float64(g.round)*0.12, hp: 1 + g.round/4})
	}
}

// spawnEdge returns a random border point, so waves close in instead of spawning on the player.
func spawnEdge() (float64, float64) {
	if rand.IntN(2) == 0 {
		return float64(rand.IntN(screenW)), float64(rand.IntN(2) * screenH)
	}
	return float64(rand.IntN(2) * screenW), float64(rand.IntN(screenH))
}

func (g *game) Update() error {
	if g.player.hp <= 0 {
		if inpututil.IsKeyJustPressed(ebiten.KeyR) {
			*g = *newGame()
		}
		return nil
	}
	g.movePlayer()
	g.fire()
	g.stepBullets()
	g.stepEnemies()
	if len(g.enemies) == 0 {
		g.nextRound()
	}
	return nil
}

// held reports 1 if any key is down, so opposing keys subtract into one movement axis.
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
	if g.cooldown > 0 {
		g.cooldown--
		return
	}
	if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		return
	}
	mx, my := ebiten.CursorPosition()
	a := math.Atan2(float64(my)-g.player.y, float64(mx)-g.player.x) // Atan2 stays defined when the cursor sits on the player
	g.bullets = append(g.bullets, ent{x: g.player.x, y: g.player.y, dx: math.Cos(a) * bulletSpeed, dy: math.Sin(a) * bulletSpeed})
	g.cooldown = fireCooldown
}

// stepBullets advances bullets, drops off-screen ones, and spends a bullet on the first
// enemy hit. Filtering in place reuses the backing array; same for stepEnemies.
func (g *game) stepBullets() {
	live := g.bullets[:0]
bullets:
	for _, b := range g.bullets {
		b.x, b.y = b.x+b.dx, b.y+b.dy
		if b.x < 0 || b.x > screenW || b.y < 0 || b.y > screenH {
			continue
		}
		for i := range g.enemies {
			if e := &g.enemies[i]; math.Hypot(e.x-b.x, e.y-b.y) < enemyRadius+bulletRadius {
				e.hp--
				continue bullets
			}
		}
		live = append(live, b)
	}
	g.bullets = live
}

// stepEnemies homes enemies on the player, banks score for the dead, and trades an enemy for HP on contact.
func (g *game) stepEnemies() {
	live := g.enemies[:0]
	for _, e := range g.enemies {
		if e.hp <= 0 {
			g.score += 10 * g.round
			continue
		}
		dx, dy := g.player.x-e.x, g.player.y-e.y
		if d := math.Hypot(dx, dy); d > 0 {
			e.x, e.y = e.x+dx/d*e.spd, e.y+dy/d*e.spd
			if d < playerRadius+enemyRadius {
				g.player.hp -= contactDamage
				continue
			}
		}
		live = append(live, e)
	}
	g.enemies = live
}

func (g *game) Draw(screen *ebiten.Image) {
	screen.Fill(colBG)
	for _, e := range g.enemies {
		vector.FillCircle(screen, float32(e.x), float32(e.y), enemyRadius, colEnemy, true)
	}
	for _, b := range g.bullets {
		vector.FillCircle(screen, float32(b.x), float32(b.y), bulletRadius, colBullet, true)
	}
	if g.player.hp > 0 {
		mx, my := ebiten.CursorPosition()
		vector.StrokeLine(screen, float32(g.player.x), float32(g.player.y), float32(mx), float32(my), 1, colAim, true)
		vector.FillCircle(screen, float32(g.player.x), float32(g.player.y), playerRadius, colPlayer, true)
	}
	label(screen, fmt.Sprintf("ROUND %d   HP %d   SCORE %d   LEFT %d", g.round, max(g.player.hp, 0), g.score, len(g.enemies)), 8, 8)
	if g.player.hp <= 0 {
		label(screen, fmt.Sprintf("GAME OVER\nreached round %d with %d points\npress R to restart", g.round, g.score), screenW/2-90, screenH/2-20)
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
