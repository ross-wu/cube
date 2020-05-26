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
	verbose   = flag.Bool("v", false, "Print the cube for each step.")
	debug     = flag.Bool("debug", false, "debug mode.")
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

	Up    byte = 'U'
	Left  byte = 'L'
	Front byte = 'F'
	Right byte = 'R'
	Back  byte = 'B'
	Down  byte = 'D'
)

var (
	faceCodes = []byte{Up, Left, Front, Right, Back, Down}
	faceNames = map[byte]string{
		Up:    "up",
		Left:  "left",
		Front: "front",
		Right: "right",
		Back:  "back",
		Down:  "down",
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
	turnChain = map[byte]byte{
		Front: Left,
		Left:  Back,
		Back:  Right,
		Right: Front,
	}
)

type Face struct {
	Pieces [9]Color
}

func FaceName(code byte) string {
	return faceNames[code]
}

type Cube struct {
	faces map[byte]*Face
	moves map[Move]func(*Cube) int

	// When flip or turn cube, the real faces change positions, but to apply the
	// solving algoritm, I'll let the virual cube steady, so use var calibs to keep
	// the mapping between virtual face to the real face.
	calibs map[byte]byte
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

func invert(f *Face) {
	f.Pieces[0], f.Pieces[8] = f.Pieces[8], f.Pieces[0]
	f.Pieces[1], f.Pieces[7] = f.Pieces[7], f.Pieces[1]
	f.Pieces[2], f.Pieces[6] = f.Pieces[6], f.Pieces[2]
	f.Pieces[3], f.Pieces[5] = f.Pieces[5], f.Pieces[3]
}

// flips the cube bottom->front->top->back->bottom.
func (c *Cube) flip() *Cube {
	last := Up
	saved := c.faces[Up]
	for _, code := range []byte{Front, Down, Back} {
		c.faces[last] = c.faces[code]
		last = code
	}
	c.faces[Back] = saved

	invert(c.faces[Back])
	invert(c.faces[Down])
	rotateClock(c.faces[Right])
	rotateCounterclock(c.faces[Left])

	// Update calibs.
	for code := range c.calibs {
		switch c.calibs[code] {
		case Up:
			c.calibs[code] = Back
		case Front:
			c.calibs[code] = Up
		case Down:
			c.calibs[code] = Front
		case Back:
			c.calibs[code] = Down
		}
	}
	if *debug {
		fmt.Printf("flip:\n")
		c.Print()
	}
	return c
}

// turns the cube front->left->back->right->front.
func (c *Cube) turn(n int) *Cube {
	for i := 0; i < n; i++ {
		last := Left
		saved := c.faces[Left]
		for _, code := range []byte{Left, Front, Right, Back} {
			c.faces[last] = c.faces[code]
			last = code
		}
		c.faces[Back] = saved

		rotateClock(c.faces[Up])
		rotateCounterclock(c.faces[Down])

		// Update calibs.
		for code := range c.calibs {
			switch c.calibs[code] {
			case Left:
				c.calibs[code] = Back
			case Front:
				c.calibs[code] = Left
			case Right:
				c.calibs[code] = Front
			case Back:
				c.calibs[code] = Right
			}
		}
	}
	if *debug {
		fmt.Printf("turn=%d:\n", n)
		c.Print()
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
	for _, cur := range []byte{Back, Right, Front} {
		for i := 6; i < 9; i++ {
			c.faces[last].Pieces[i] = c.faces[cur].Pieces[i]
		}
		last = cur
	}
	for i := 0; i < 3; i++ {
		c.faces[last].Pieces[i+6] = saved[i]
	}

	rotateClock(c.faces[Down])
	return c
}

func (c *Cube) D2() *Cube {
	return c.D().D()
}

func (c *Cube) d() *Cube {
	return c.D().D().D()
}

func newMoves() map[Move]func(*Cube) int {
	// Basic moves.
	m := map[Move]func(*Cube) int{
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
		faces: map[byte]*Face{},
		moves: newMoves(),
		calibs: map[byte]byte{
			Up:    Up,
			Left:  Left,
			Front: Front,
			Right: Right,
			Back:  Back,
			Down:  Down,
		},
	}
}

func (c *Cube) Calib(old Move) Move {
	b := []byte(old)
	b[0] = c.calibs[b[0]]
	return Move(b)
}

// Rotate rotates cube fase according to the given move. The move should be the virtual face.
func (c *Cube) Rotate(m Move) error {
	op, ok := c.moves[c.Calib(m)]
	if !ok {
		return fmt.Errorf("no such move: %s", m)
	}
	op(c)
	return nil
}

func (c *Cube) KociembaScramble() string {
	colorToCode := map[Color]byte{}
	for code, face := range c.faces {
		colorToCode[face.Pieces[4]] = c.calibs[code]
	}
	buf := make([]byte, 0, 55)
	for _, name := range []byte{Up, Right, Front, Down, Left, Back} {
		for _, color := range c.faces[name].Pieces {
			buf = append(buf, colorToCode[color])
		}
	}
	return string(buf)
}

func (c *Cube) CalibsDebugString() string {
	s := "{"
	for _, from := range []byte{Up, Left, Front, Right, Back, Down} {
		to := c.calibs[from]
		s += fmt.Sprintf(" %c->%c", from, to)
	}
	s += " }"
	return s
}

func (c *Cube) Apply(moves []string, printStep bool) error {
	for i := range moves {
		m := Move(strings.TrimSpace(moves[i]))
		if len(m) == 0 {
			continue
		}
		if *verbose {
			fmt.Printf("calibs: %s\n", c.CalibsDebugString())
		}
		if err := c.Rotate(m); err != nil {
			log.Printf("ERROR: Rotate(%s) error: %v", string(m), err)
			return err
		}
		if printStep {
			fmt.Printf("Step[%d]: newMove=%s\n", i+1, m)
			c.Print()
		}
	}
	return nil
}

func (c *Cube) SetFace(code byte, face *Face) {
	c.faces[code] = face
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
		printRow(c.faces[Up], row)
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
		printRow(c.faces[Down], row)
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
	s := c.KociembaScramble()
	log.Printf("INFO: exec: %s %s", *kociemba, s)
	out, err := exec.Command(*kociemba, s).Output()
	if err != nil {
		log.Printf("ERROR: failed to run %q: %v", *kociemba, err)
		os.Exit(255)
	}
	return strings.Split(strings.TrimSpace(string(out)), " ")
}

func main() {
	flag.Parse()

	fmt.Println("Please set colors of each pieces on each face.")
	fmt.Println("Colors are: White Red Green Blue Yellow Orange, or w r g b y o.")
	fmt.Println("(input 9 whitespace-separated colors for each face):")

	if *debug {
		*verbose = true
	}

	c := NewCube()
	reader := bufio.NewReader(os.Stdin)
	for _, code := range faceCodes {
		c.SetFace(code, readFace(FaceName(code), reader))
	}

	fmt.Printf("\n\nINPUT:\n")
	c.Print()

	if len(*initMoves) > 0 {
		fmt.Println("------------------------------------------------------")
		fmt.Printf("INITIAL MOVES:\n")
		for _, m := range strings.Split(*initMoves, ",") {
			fmt.Printf("\nmove=%s (->%s):\n", m, c.Calib(Move(m)))
			fmt.Printf("calibs: %s\n", c.CalibsDebugString())
			c.Rotate(Move(m))
			c.Print()
		}
		os.Exit(0)
	}

	steps := solve(c)
	fmt.Println("------------------------------------------------------")
	fmt.Printf("SOLUTION: step=%d: %s\n", len(steps), strings.Join(steps, " "))

	if err := c.Apply(steps, *verbose); err != nil {
		fmt.Printf("ERROR: Apply(%v) error: %v", steps, err)
		os.Exit(255)
	}
	if !*verbose {
		c.Print()
	}

	fmt.Printf("\nDONE\n")
}
