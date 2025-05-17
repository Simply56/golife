package main

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/veandco/go-sdl2/sdl"
)

type Protocol int

const (
	Off Protocol = iota
	DenseCells
	SparsePixels
	DensePixels
)

const (
	PROTOCOL   = Off
	VISUAL_OUT = true
	EMPTY      = 0
	BLUE       = 1
	ORANGE     = 2
	DEAD       = 3
	gridWidth  = 500
	gridHeight = 500
)

type Game struct {
	grid     [][]uint8
	nextGrid [][]uint8
}

// NewGame creates a new Game of Life with a random initial state
func NewGame() *Game {
	grid := make([][]uint8, gridWidth)
	nextGrid := make([][]uint8, gridWidth)
	for i := range grid {
		grid[i] = make([]uint8, gridHeight)
		nextGrid[i] = make([]uint8, gridHeight)
		for j := range grid[i] {
			grid[i][j] = uint8(rand.Int()) % 3
		}
	}
	return &Game{
		grid:     grid,
		nextGrid: nextGrid,
	}
}

func (g *Game) Swap() {
	// Swap current and next generation
	g.grid, g.nextGrid = g.nextGrid, g.grid
}

// CountNeighbors counts the number of live neighbors for a cell
func (g *Game) CountNeighbors(x, y int) (blue_count, orange_count int) {
	for i := -1; i <= 1; i++ {
		for j := -1; j <= 1; j++ {
			if i == 0 && j == 0 {
				continue // Skip the cell itself
			}

			// Wrap around the edges
			nx := (x + i + gridWidth) % gridWidth
			ny := (y + j + gridHeight) % gridHeight
			switch g.grid[nx][ny] {
			case BLUE:
				blue_count++
			case ORANGE:
				orange_count++
			}
		}
	}
	return
}

// Update advances the game to the next generation
func (g *Game) Update() {

	numCPU := runtime.NumCPU()
	var wg sync.WaitGroup

	// Divide work based on CPU cores
	rowsPerWorker := gridWidth / numCPU
	for i := range numCPU {
		wg.Add(1)
		startRow := i * rowsPerWorker
		endRow := startRow + rowsPerWorker
		if i == numCPU-1 {
			endRow = gridWidth // Handle remainder
		}

		go func(startRow, endRow int) {
			defer wg.Done()
			for x := startRow; x < endRow; x++ {
				for y := range gridHeight {

					cell := g.grid[x][y]

					// Handle dead
					if g.grid[x][y] >= DEAD {
						if g.grid[x][y] == 6 {
							g.nextGrid[x][y] = EMPTY
							continue
						}
						g.nextGrid[x][y] = g.grid[x][y] + 1
						continue
					}

					blue_count, orange_count := g.CountNeighbors(x, y)
					count := blue_count + orange_count

					// Handle living
					if g.grid[x][y] == BLUE || g.grid[x][y] == ORANGE {
						if !(3 > count) && !(count > 5) {
							g.nextGrid[x][y] = DEAD
							continue
						}
						g.nextGrid[x][y] = cell
						continue
					}

					// Handle empty
					if (cell == EMPTY) && (count == 3) {
						if blue_count > orange_count {
							g.nextGrid[x][y] = BLUE
							continue
						}
						g.nextGrid[x][y] = ORANGE
						continue
					}
					// Empty stays empty
					g.nextGrid[x][y] = cell
				}
			}
		}(startRow, endRow)
	}

	wg.Wait()
}

// Draw renders the current state of the game to an SDL texture
func (g *Game) Draw(renderer *sdl.Renderer) {
	// Prepare point groups per color
	var bluePoints []sdl.Point
	var orangePoints []sdl.Point
	var blackPoints []sdl.Point
	var darkPoints []sdl.Point
	var greyPoints []sdl.Point

	bluePoints = make([]sdl.Point, 0, gridWidth*gridHeight/8)
	orangePoints = make([]sdl.Point, 0, gridWidth*gridHeight/8)
	blackPoints = make([]sdl.Point, 0, gridWidth*gridHeight/8)
	darkPoints = make([]sdl.Point, 0, gridWidth*gridHeight/8)
	greyPoints = make([]sdl.Point, 0, gridWidth*gridHeight/8)

	// Collect points by color
	for x := range gridWidth {
		for y := range gridHeight {
			switch g.grid[x][y] {
			case ORANGE:
				orangePoints = append(orangePoints, sdl.Point{X: int32(x), Y: int32(y)})
			case BLUE:
				bluePoints = append(bluePoints, sdl.Point{X: int32(x), Y: int32(y)})
			case DEAD:
				bluePoints = append(bluePoints, sdl.Point{X: int32(x), Y: int32(y)})
			case DEAD + 1:
				bluePoints = append(bluePoints, sdl.Point{X: int32(x), Y: int32(y)})
			case DEAD + 2:
				bluePoints = append(bluePoints, sdl.Point{X: int32(x), Y: int32(y)})
			case DEAD + 3:
				bluePoints = append(bluePoints, sdl.Point{X: int32(x), Y: int32(y)})
			}
		}
	}

	// Draw each color group in batches
	if len(bluePoints) > 0 {
		renderer.SetDrawColor(0x0, 0x99, 0xFF, 0xFF)
		renderer.DrawPoints(bluePoints)
	}
	if len(orangePoints) > 0 {
		renderer.SetDrawColor(0xFF, 0x99, 0x00, 0xFF)
		renderer.DrawPoints(orangePoints)
	}
	if len(blackPoints) > 0 {
		renderer.SetDrawColor(0x66, 0x66, 0x66, 0xFF)
		renderer.DrawPoints(blackPoints)
	}
	if len(darkPoints) > 0 {
		renderer.SetDrawColor(0x7f, 0x7f, 0x7f, 0xFF)
		renderer.DrawPoints(darkPoints)
	}

	if len(greyPoints) > 0 {
		renderer.SetDrawColor(0x99, 0x99, 0x99, 0xFF)
		renderer.DrawPoints(greyPoints)
	}
}

func (g *Game) outputSparsePixels() error {
	rowData := make([]byte, 4*gridWidth)
	for y := range gridHeight {
		rowData = rowData[:0]
		for x := range gridWidth {
			state := g.grid[x][y]
			if state == EMPTY {
				continue
			}

			// Pack x (12 bits), y (12 bits), state (8 bits)
			packed := uint32(x&0xFFF) | uint32((y&0xFFF)<<12) | (uint32(state) << 24)

			var cell [4]byte
			binary.LittleEndian.PutUint32(cell[:], packed)

			rowData = append(rowData, cell[:]...)
		}

		_, err := os.Stdout.Write(rowData)
		if err != nil {
			return err
		}
	}

	// End-of-frame marker
	var eof [4]byte
	binary.LittleEndian.PutUint32(eof[:], 0xFFFFFFFF)
	_, err := os.Stdout.Write(eof[:])
	return err
}

func rgba(r, g, b uint8) uint32 {
	return uint32(b) | uint32(g)<<8 | uint32(r)<<16
}
func (g *Game) ouputDensePixels() error {
	// Create a buffer for a single row
	row := make([]byte, gridWidth*4)

	for y := range gridHeight {
		for x := range gridWidth {
			var color uint32
			switch g.grid[x][y] {
			case BLUE:
				color = rgba(0, 0, 255)
			case ORANGE:
				color = rgba(255, 128, 0)
			case DEAD:
				color = rgba(0, 0, 0)
			case DEAD + 1:
				color = rgba(136, 136, 136)
			case DEAD + 2:
				color = rgba(160, 160, 160)
			case DEAD + 3:
				color = rgba(238, 238, 238)
			default:
				color = rgba(255, 255, 255)
			}
			binary.LittleEndian.PutUint32(row[x*4:], color)
		}
		_, err := os.Stdout.Write(row)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) ouputDenseCells() error {
	for y := range gridHeight {
		_, err := os.Stdout.Write(g.grid[y])
		if err != nil {
			return err
		}
	}
	return nil
}

func (game *Game) visualize() {
	var renderer *sdl.Renderer

	// Initialize SDL
	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	// Create window
	window, err := sdl.CreateWindow(
		"Conway's Game of Life",
		sdl.WINDOWPOS_UNDEFINED, sdl.WINDOWPOS_UNDEFINED,
		gridWidth, gridHeight,
		sdl.WINDOW_SHOWN,
	)
	if err != nil {
		panic(err)
	}
	defer window.Destroy()

	// Create renderer
	renderer, err = sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		panic(err)
	}
	defer renderer.Destroy()

	for {
		// Poll for events to keep the window responsive
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch e := event.(type) {
			case *sdl.QuitEvent:
				return // exit the program cleanly
			case *sdl.KeyboardEvent:
				if e.Keysym.Sym == sdl.K_ESCAPE && e.State == sdl.PRESSED {
					return
				}
			}
		}

		printFPS()

		// Clear the screen with white
		renderer.SetDrawColor(255, 255, 255, 255)
		renderer.Clear()

		game.Update()
		// Draw the game
		game.Draw(renderer)

		// Update the screen
		renderer.Present()

		game.Swap()
	}
}
func (g *Game) present() {
	switch PROTOCOL {
	case DensePixels:
		g.ouputDensePixels()
	case DenseCells:
		g.ouputDenseCells()
	case SparsePixels:
		g.outputSparsePixels()
	}
}

func main() {
	game := NewGame()
	if VISUAL_OUT {
		game.visualize()
	}

	for {
		var wg sync.WaitGroup
		wg.Add(1)
		// Print the binary output
		go func() {
			defer wg.Done()
			game.present()
		}()

		// Update game state
		wg.Add(1)
		go func() {
			defer wg.Done()
			game.Update()
		}()
		wg.Wait()
		game.Swap()
	}
}

// Helper function
// Package level variables for FPS counter
var (
	fpsCounter     int
	fpsLastPrint   time.Time
	fpsInitialized bool
)

func printFPS() {
	// Initialize on first call
	if !fpsInitialized {
		fpsLastPrint = time.Now()
		fpsInitialized = true
	}

	// Increment the counter
	fpsCounter++

	// Check if a second has passed
	now := time.Now()
	if now.Sub(fpsLastPrint).Seconds() >= 1.0 {
		fmt.Fprintf(os.Stderr, "FPS: %d\n", fpsCounter)
		fpsCounter = 0
		fpsLastPrint = now
	}
}
