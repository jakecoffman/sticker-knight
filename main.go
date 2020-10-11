package main

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/jakecoffman/sticker-knight/camera"
	"github.com/jakecoffman/sticker-knight/input"
	"github.com/jakecoffman/sticker-knight/tiled"
	"golang.org/x/image/math/f64"
	"log"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	. "github.com/jakecoffman/cp"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	ebiten.SetVsyncEnabled(false)

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Sticker Knight")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}

const (
	screenWidth, screenHeight = 800, 600

	playerVelocity = 500.0

	playerGroundAccelTime = 0.1
	playerGroundAccel     = playerVelocity / playerGroundAccelTime

	playerAirAccelTime = 0.25
	playerAirAccel     = playerVelocity / playerAirAccelTime

	jumpHeight      = 50.0
	jumpBoostHeight = 55.0
	fallVelocity    = 900.0
	gravity         = 2000.0
)

type Game struct {
	map1 *tiled.Map

	camera *camera.Camera
	world *ebiten.Image

	drawPhysics bool

	space       *Space
	playerBody  *Body
	playerShape *Shape

	remainingBoost          float64
	grounded, lastJumpState bool
}

func NewGame() *Game {
	game := &Game{
		camera: &camera.Camera{
			ViewPort:   f64.Vec2{screenWidth, screenHeight},
			Position:   f64.Vec2{861, 766},
			ZoomFactor: -106,
			Rotation:   0,
		},
	}
	space := NewSpace()
	space.Iterations = 10
	space.SetGravity(Vector{0, -gravity})

	walls := []Vector{
		{-320, -240}, {-320, 240},
		{320, -240}, {320, 240},
		{-320, -240}, {320, -240},
		{-320, 240}, {320, 240},
	}
	for i := 0; i < len(walls)-1; i += 2 {
		shape := space.AddShape(NewSegment(space.StaticBody, walls[i], walls[i+1], 0))
		shape.SetElasticity(1)
		shape.SetFriction(1)
		shape.SetFilter(input.NotGrabbable)
	}

	// player
	playerBody := space.AddBody(NewBody(1, INFINITY))
	playerBody.SetPosition(Vector{0, -200})
	playerBody.SetVelocityUpdateFunc(game.playerUpdateVelocity)

	playerShape := space.AddShape(NewBox2(playerBody, BB{-15, -27.5, 15, 27.5}, 10))
	playerShape.SetElasticity(0)
	playerShape.SetFriction(0)
	playerShape.SetCollisionType(1)

	for i := 0; i < 6; i++ {
		for j := 0; j < 3; j++ {
			body := space.AddBody(NewBody(4, INFINITY))
			body.SetPosition(Vector{float64(100 + j*60), float64(-200 + i*60)})

			shape := space.AddShape(NewBox(body, 50, 50, 0))
			shape.SetElasticity(0)
			shape.SetFriction(0.7)
		}
	}

	game.map1 = tiled.NewMap("sandbox")
	game.world = ebiten.NewImage(game.map1.Width*game.map1.TileWidth, game.map1.Height*game.map1.TileHeight)
	game.space = space
	game.playerBody = playerBody
	game.playerShape = playerShape
	return game
}

var lastFrameTime time.Time

func (g *Game) Update() error {
	currentTime := time.Now()
	dt := currentTime.Sub(lastFrameTime).Seconds()
	lastFrameTime = currentTime

	if ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyLeft) {
		g.camera.Position[0] -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyRight) {
		g.camera.Position[0] += 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyUp) {
		g.camera.Position[1] -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyDown) {
		g.camera.Position[1] += 1
	}

	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		g.camera.ZoomFactor -= 1
	}
	if ebiten.IsKeyPressed(ebiten.KeyE) {
		g.camera.ZoomFactor += 1
	}

	if ebiten.IsKeyPressed(ebiten.KeyR) {
		g.camera.Rotation += 1
	}

	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		g.camera.Reset()
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.drawPhysics = !g.drawPhysics
	}

	jumpState := ebiten.IsKeyPressed(ebiten.KeySpace)

	// If the jump key was just pressed this frame, jump!
	if jumpState && !g.lastJumpState && g.grounded {
		jumpV := math.Sqrt(2.0 * jumpHeight * gravity)
		g.playerBody.SetVelocityVector(g.playerBody.Velocity().Add(Vector{0, jumpV}))

		g.remainingBoost = jumpBoostHeight / jumpV
	}

	g.space.Step(1. / 180.)

	g.remainingBoost -= dt
	g.lastJumpState = jumpState
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.world.Clear()

	for _, group := range g.map1.ObjectGroups {
		for _, object := range group.Objects {
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(object.X, object.Y)

			img, ok := g.map1.Tiles[object.GID]
			// missing GID means there's no sprite, like for map bounds
			if ok {
				g.world.DrawImage(img, op)
			}
		}
	}

	g.camera.Render(g.world, screen)

	worldX, worldY := g.camera.ScreenToWorld(ebiten.CursorPosition())
	ebitenutil.DebugPrint(
		screen,
		fmt.Sprintf("FPS: %0.2f\nMove (WASD/Arrows)\nZoom (QE)\nRotate (R)\nReset (Space)", ebiten.CurrentFPS()),
	)
	ebitenutil.DebugPrintAt(
		screen,
		fmt.Sprintf("%s\nCursor World Pos: %.2f,%.2f",
			g.camera.String(),
			worldX, worldY),
		0, screenHeight-32,
	)
}

func (g *Game) Layout(int, int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) playerUpdateVelocity(body *Body, gravityV Vector, damping, dt float64) {
	var jumpState bool
	if ebiten.IsKeyPressed(ebiten.KeySpace) {
		jumpState = true
	}

	// Grab the grounding normal from last frame
	groundNormal := Vector{}
	g.playerBody.EachArbiter(func(arb *Arbiter) {
		n := arb.Normal().Neg()

		if n.Y > groundNormal.Y {
			groundNormal = n
		}
	})

	g.grounded = groundNormal.Y > 0
	if groundNormal.Y < 0 {
		g.remainingBoost = 0
	}

	// Do a normal-ish update
	boost := jumpState && g.remainingBoost > 0
	var grav Vector
	if !boost {
		grav = gravityV
	}
	body.UpdateVelocity(grav, damping, dt)

	// Target horizontal speed for air/ground control
	var targetVx float64
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		targetVx += playerVelocity
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		targetVx -= playerVelocity
	}

	// Update the surface velocity and friction
	// Note that the "feet" move in the opposite direction of the player.
	surfaceV := Vector{-targetVx, 0}
	g.playerShape.SetSurfaceV(surfaceV)
	if g.grounded {
		g.playerShape.SetFriction(playerGroundAccel / gravity)
	} else {
		g.playerShape.SetFriction(0)
	}

	// Apply air control if not grounded
	if !g.grounded {
		v := g.playerBody.Velocity()
		g.playerBody.SetVelocity(LerpConst(v.X, targetVx, playerAirAccel*dt), v.Y)
	}

	v := body.Velocity()
	body.SetVelocity(v.X, Clamp(v.Y, -fallVelocity, INFINITY))
}
