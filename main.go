// This binary is a Rubik's cube resolver.
// Algorithm and move notations are described in
// https://cube3x3.com/how-to-solve-a-rubiks-cube/
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

var (
	port      = flag.Int("port", 8080, "http server port")
	kociemba  = flag.String("kociemba", "./kociemba/bin/kociemba", "Path to the Kociemba's Rubik's Cube solver binary.")
	initMoves = flag.String("init_move", "", "Comma-separated initial moves, for test only.")
)

type Color int
type Move string

const (
	White Color = iota
	Red
	Green
	Blue
	Yellow
	Orange
	Unknown

	Top    = "top"
	Left   = "left"
	Front  = "front"
	Right  = "right"
	Back   = "back"
	Bottom = "bottom"
)

var (
	faceNames = []string{Top, Left, Front, Right, Back, Bottom}
	faceCode  = map[string]byte{
		Top:    'U',
		Left:   'L',
		Front:  'F',
		Right:  'R',
		Back:   'B',
		Bottom: 'D',
	}
	colors = map[Color]string{
		White:   "\033[37m",
		Red:     "\033[31m",
		Green:   "\033[32m",
		Blue:    "\033[34m",
		Yellow:  "\033[33m",
		Orange:  "\033[1;31m",
		Unknown: "\033[30m",
	}
	leftNeighbor = map[string]string{
		Left:  Back,
		Front: Left,
		Right: Front,
		Back:  Right,
	}
	rightNeighbor = map[string]string{
		Left:  Front,
		Front: Right,
		Right: Back,
		Back:  Left,
	}
	topNeighbor    = map[string]string{}
	bottomNeighbor = map[string]string{}
)

type Face struct {
	Pieces [9]Color
}

type Cube struct {
	faces map[string]*Face
	moves map[string]func(*Cube) int

	// When flip or turn cube, the real faces change positions, but to apply the
	// solving algoritm, I'll the virual cube steady, so use var calibs to keep
	// the mapping between virtual face to the real face.
	calibs map[string]string
}

func (c *Cube) kociembaScramble() string {
	colorToCode := map[Color]byte{}
	for name, face := range c.faces {
		colorToCode[face.Pieces[4]] = faceCode[name]
	}
	buf := make([]byte, 0, 55)
	for _, name := range []string{Top, Right, Front, Bottom, Left, Back} {
		for _, color := range c.faces[name].Pieces {
			buf = append(buf, colorToCode[color])
		}
	}
	return string(buf)
}

func (c *Cube) Face(name string) *Face {
	return c.faces[c.calibs[name]]
}

func rotateClock(f *Face) {
	// Move corners.
	saved := f.Pieces[0]
	f.Pieces[0] = f.Pieces[6]
	f.Pieces[6] = f.Pieces[8]
	f.Pieces[8] = f.Pieces[2]
	f.Pieces[2] = saved

	// Move edges.
	saved = f.Pieces[1]
	f.Pieces[1] = f.Pieces[3]
	f.Pieces[3] = f.Pieces[7]
	f.Pieces[7] = f.Pieces[5]
	f.Pieces[5] = saved
}

func rotateCounterclock(f *Face) {
	// Move corners.
	saved := f.Pieces[0]
	f.Pieces[0] = f.Pieces[2]
	f.Pieces[2] = f.Pieces[8]
	f.Pieces[8] = f.Pieces[6]
	f.Pieces[6] = saved

	// Move edges.
	saved = f.Pieces[1]
	f.Pieces[1] = f.Pieces[5]
	f.Pieces[5] = f.Pieces[7]
	f.Pieces[7] = f.Pieces[3]
	f.Pieces[3] = saved
}

// flips the cube bottom->front->top->back->bottom.
func (c *Cube) flip() *Cube {
	last := Top
	saved := c.faces[Top]
	for _, name := range []string{Front, Bottom, Bottom, Back} {
		c.faces[last] = c.faces[name]
		last = name
	}
	c.faces[Back] = saved

	rotateClock(c.faces[Right])
	rotateCounterclock(c.faces[Left])

	c.calibs[Top] = Back
	c.calibs[Back] = Bottom
	c.calibs[Bottom] = Front
	c.calibs[Front] = Top
	return c
}

func (c *Cube) reverseFlip() *Cube {
	return c.flip().flip().flip()
}

// turns the cube front->left->back->right->front.
func (c *Cube) turn(n int) *Cube {
	for i := 0; i < n; i++ {
		last := Left
		saved := c.faces[Left]
		for _, name := range []string{Left, Front, Right, Back} {
			c.faces[last] = c.faces[name]
			last = name
		}
		c.faces[Back] = saved

		rotateClock(c.faces[Top])
		rotateCounterclock(c.faces[Bottom])

		c.calibs[Front] = Left
		c.calibs[Left] = Back
		c.calibs[Back] = Right
		c.calibs[Right] = Front
	}
	return c
}

func (c *Cube) reverseTurn() *Cube {
	return c.turn(3)
}

// D performs a D move on the cube.
func (c *Cube) D() *Cube {
	last := Left
	left := c.faces[Left].Pieces
	saved := [3]Color{left[6], left[7], left[8]}
	for _, cur := range []string{Front, Right, Back} {
		for i := 6; i < 9; i++ {
			c.faces[last].Pieces[i] = c.faces[cur].Pieces[i]
		}
		last = cur
	}
	for i := 0; i < 3; i++ {
		c.faces[last].Pieces[i+6] = saved[i]
	}

	rotateClock(c.faces[Bottom])
	return c
}

func (c *Cube) D2() *Cube {
	return c.D().D()
}

func (c *Cube) d() *Cube {
	return c.D().D().D()
}

func newMoves() map[string]func(*Cube) int {
	// Basic moves.
	m := map[string]func(*Cube) int{
		"D": func(c *Cube) int {
			c.D()
			return 1
		},
		"D2": func(c *Cube) int {
			c.D().D()
			return 1
		},
		"D'": func(c *Cube) int {
			c.d()
			return 1
		},

		"B": func(c *Cube) int {
			c.flip().D()
			return 2
		},
		"B2": func(c *Cube) int {
			c.flip().D2()
			return 2
		},
		"B'": func(c *Cube) int {
			c.flip().d()
			return 2
		},

		"U": func(c *Cube) int {
			c.flip().flip().D()
			return 3
		},
		"U2": func(c *Cube) int {
			c.flip().flip().D2()
			return 3
		},
		"U'": func(c *Cube) int {
			c.flip().flip().d()
			return 3
		},

		"L": func(c *Cube) int {
			c.turn(1).flip().D()
			return 3
		},
		"L2": func(c *Cube) int {
			c.turn(1).flip().D2()
			return 3
		},
		"L'": func(c *Cube) int {
			c.turn(1).flip().d()
			return 3
		},

		"R": func(c *Cube) int {
			c.reverseTurn().flip().D()
			return 3
		},
		"R2": func(c *Cube) int {
			c.reverseTurn().flip().D2()
			return 3
		},
		"R'": func(c *Cube) int {
			c.reverseTurn().flip().d()
			return 3
		},

		"F": func(c *Cube) int {
			c.turn(2).flip().D()
			return 3
		},
		"F2": func(c *Cube) int {
			c.turn(2).flip().D2()
			return 3
		},
		"F'": func(c *Cube) int {
			c.turn(2).flip().d()
			return 3
		},
	}

	return m
}

func NewCube() *Cube {
	return &Cube{
		faces: map[string]*Face{},
		moves: newMoves(),
		calibs: map[string]string{
			Top:    Top,
			Left:   Left,
			Front:  Front,
			Right:  Right,
			Back:   Back,
			Bottom: Bottom,
		},
	}
}

// Rotate rotates cube fase according to the given move.
func (c *Cube) Rotate(m Move) error {
	op, ok := c.moves[string(m)]
	if !ok {
		return fmt.Errorf("no such move: %s", m)
	}
	op(c)
	return nil
}

func (c *Cube) SetFace(name string, face *Face) {
	c.faces[name] = face
}

func printColor(color Color) {
	fmt.Printf("%s▇▇\033[0m", colors[color])
}

func printPiece(face *Face, row, col int) {
	printColor(face.Pieces[row*3+col])
}

func printRow(face *Face, row int) {
	for col := 0; col < 3; col++ {
		if col == 0 {
			fmt.Printf("│")
		} else {
			fmt.Print(" ")
		}
		printPiece(face, row, col)
	}
}

func (c *Cube) Print() {
	const indent = "         "

	fmt.Printf("\t         ┌────────┐\n")

	// Print top face.
	for row := 0; row < 3; row++ {
		fmt.Printf("\t%s", indent)
		printRow(c.faces[Top], row)
		fmt.Printf("│\n")
	}

	fmt.Printf("\t┌────────┼────────┼────────┬────────┐\n")

	// Print left, front, right and back faces
	for row := 0; row < 3; row++ {
		fmt.Printf("\t")
		printRow(c.faces[Left], row)
		printRow(c.faces[Front], row)
		printRow(c.faces[Right], row)
		printRow(c.faces[Back], row)
		fmt.Printf("│\n")
	}

	fmt.Printf("\t└────────┼────────┼────────┴────────┘\n")

	// Print bottom face.
	for row := 0; row < 3; row++ {
		fmt.Printf("\t%s", indent)
		printRow(c.faces[Bottom], row)
		fmt.Printf("│\n")
	}

	fmt.Printf("\t         └────────┘\n")
}

func parseField(s string) Color {
	s = strings.ToLower(s)
	switch s {
	case "white", "w":
		return White
	case "Red", "r":
		return Red
	case "Green", "g":
		return Green
	case "Blue", "b":
		return Blue
	case "Yellow", "y":
		return Yellow
	case "Orange", "o":
		return Orange
	}
	return Unknown
}

func readFace(faceName string, reader *bufio.Reader) *Face {
	fmt.Printf("\n  %s: ", faceName)

	face := &Face{}
	i := 0
	for i < 9 {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("ERROR: ReadString error: %v", err)
			os.Exit(1)
		}
		for _, s := range strings.Fields(line) {
			if i == 9 {
				fmt.Println("WARNING: read 9 pieces already, discards rest of the input line.")
				break
			}
			face.Pieces[i] = parseField(s)
			i++
		}
		if i < 9 {
			fmt.Printf("%d remain> ", 9-i)
		}
	}
	return face
}

func solve(c *Cube) []string {
	s := c.kociembaScramble()
	log.Printf("INFO: exec: %s %s", *kociemba, s)
	out, err := exec.Command(*kociemba, s).Output()
	if err != nil {
		log.Printf("ERROR: failed to run %q: %v", *kociemba, err)
		os.Exit(255)
	}
	return strings.Split(string(out), " ")
}

func main() {
	flag.Parse()

	fmt.Println("Please set colors of each pieces on each face.")
	fmt.Println("Colors are: White Red Green Blue Yellow Orange, or w r g b y o.")
	fmt.Println("(input 9 whitespace-separated colors for each face):")

	c := NewCube()
	reader := bufio.NewReader(os.Stdin)
	for _, faceName := range faceNames {
		c.SetFace(faceName, readFace(faceName, reader))
	}

	fmt.Printf("\n\nINPUT:\n")
	c.Print()

	if len(*initMoves) > 0 {
		for _, m := range strings.Split(*initMoves, ",") {
			c.Rotate(Move(m))
		}
		fmt.Println("------------------------------------------------------")
		fmt.Printf("AFTER INITIAL MOVES:\n")
		c.Print()
	}

	steps := solve(c)
	fmt.Println("------------------------------------------------------")
	fmt.Printf("SOLUTION: total steps: %d: %s\n", len(steps), strings.Join(steps, " "))
}
