// This binary is a Rubik's cube resolver server.
//
// Usage:
//   $ ./server
//   then, in brower:
//     http://localhost/cube?U=yyoyygbwo&L=ggwooboob&F=rrwybwyoo&R=brgbrgyrg&B=wrrwgywoy&D=rbbgwbgwr
//
// The algorithm and move notations are described in
// https://cube3x3.com/how-to-solve-a-rubiks-cube/
//
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var (
	port     = flag.Int("port", 80, "http server port")
	kociemba = flag.String("kociemba", "./kociemba/bin/kociemba", "Path to the Kociemba's Rubik's Cube solver binary.")
	verbose  = flag.Bool("v", false, "Print the cube for each step.")
	debug    = flag.Bool("debug", false, "debug mode.")
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

	MoveFlip  = "flip"
	MoveTurn1 = "turn"
	MoveTurn2 = "turn2"
	MoveRTurn = "turn'"
	MoveD     = "D"
	MoveD2    = "D2"
	Moved     = "D'"
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
	moves map[Move]func(*Cube) []string

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

func newMoves() map[Move]func(*Cube) []string {
	// Basic moves.
	m := map[Move]func(*Cube) []string{
		"D": func(c *Cube) []string {
			c.D()
			return []string{MoveD}
		},
		"D2": func(c *Cube) []string {
			c.D().D()
			return []string{MoveD2}
		},
		"D'": func(c *Cube) []string {
			c.d()
			return []string{Moved}
		},

		"B": func(c *Cube) []string {
			c.flip().D()
			return []string{MoveFlip, MoveD}
		},
		"B2": func(c *Cube) []string {
			c.flip().D2()
			return []string{MoveFlip, MoveD2}
		},
		"B'": func(c *Cube) []string {
			c.flip().d()
			return []string{MoveFlip, Moved}
		},

		"U": func(c *Cube) []string {
			c.flip().flip().D()
			return []string{MoveFlip, MoveFlip, MoveD}
		},
		"U2": func(c *Cube) []string {
			c.flip().flip().D2()
			return []string{MoveFlip, MoveFlip, MoveD2}
		},
		"U'": func(c *Cube) []string {
			c.flip().flip().d()
			return []string{MoveFlip, MoveFlip, Moved}
		},

		"L": func(c *Cube) []string {
			c.turn(1).flip().D()
			return []string{MoveTurn1, MoveFlip, MoveD}
		},
		"L2": func(c *Cube) []string {
			c.turn(1).flip().D2()
			return []string{MoveTurn1, MoveFlip, MoveD2}
		},
		"L'": func(c *Cube) []string {
			c.turn(1).flip().d()
			return []string{MoveTurn1, MoveFlip, Moved}
		},

		"R": func(c *Cube) []string {
			c.reverseTurn().flip().D()
			return []string{MoveRTurn, MoveFlip, MoveD}
		},
		"R2": func(c *Cube) []string {
			c.reverseTurn().flip().D2()
			return []string{MoveRTurn, MoveFlip, MoveD2}
		},
		"R'": func(c *Cube) []string {
			c.reverseTurn().flip().d()
			return []string{MoveRTurn, MoveFlip, Moved}
		},

		"F": func(c *Cube) []string {
			c.turn(2).flip().D()
			return []string{MoveTurn2, MoveFlip, MoveD}
		},
		"F2": func(c *Cube) []string {
			c.turn(2).flip().D2()
			return []string{MoveTurn2, MoveFlip, MoveD2}
		},
		"F'": func(c *Cube) []string {
			c.turn(2).flip().d()
			return []string{MoveTurn2, MoveFlip, Moved}
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
func (c *Cube) Rotate(m Move) ([]string, error) {
	op, ok := c.moves[c.Calib(m)]
	if !ok {
		return nil, fmt.Errorf("no such move: %s", m)
	}
	return op(c), nil
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

// Apply applys the solution and output the physical movements.
func (c *Cube) Apply(moves []string, printStep bool) ([]string, error) {
	pMoves := []string{}
	for i := range moves {
		m := Move(strings.TrimSpace(moves[i]))
		if len(m) == 0 {
			continue
		}
		if *verbose {
			fmt.Printf("calibs: %s\n", c.CalibsDebugString())
		}
		s, err := c.Rotate(m)
		if err != nil {
			log.Printf("ERROR: Rotate(%s) error: %v", string(m), err)
			return nil, err
		}
		pMoves = append(pMoves, s...)
		if printStep {
			fmt.Printf("Step[%d]: newMove=%s %v\n", i+1, m, s)
			c.Print()
		}
	}
	return pMoves, nil
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

func readFace(faceName, line string) (*Face, error) {
	face := &Face{}
	i := 0
	for _, b := range line {
		s := fmt.Sprintf("%c", b)
		face.Pieces[i] = parseField(s)
		if face.Pieces[i] == Unknown {
			return nil, fmt.Errorf("unknown color name: %s", s)
		}
		i++
	}
	if i < 9 {
		return nil, fmt.Errorf("face string must contain 9 chars, but only had %d", 9-i)
	}
	log.Printf("readFace %s ok.", faceName)
	return face, nil
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

func httpCube(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	log.Printf("%s: %s %s", req.RemoteAddr, req.Method, req.URL.Path)

	c := NewCube()
	for _, code := range []byte{Up, Left, Front, Right, Back, Down} {
		k := fmt.Sprintf("%c", code)
		v := req.FormValue(k)
		if len(v) != 9 {
			log.Printf("ERROR: invalid arg %s=%s", k, v)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf(`face %s must contain only [wrboyg], and must be 9 chars.`, k)))
			return
		}
		face, err := readFace(k, v)
		if err != nil {
			msg := fmt.Sprintf("ERROR: readFace(%s, %s) error: %v", k, v, err)
			log.Print(msg)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(msg))
			return
		}
		c.SetFace(code, face)
	}
	fmt.Println("Input cube:")
	c.Print()

	steps := solve(c)
	solution := fmt.Sprintf("step=%d: %s", len(steps), strings.Join(steps, " "))
	log.Printf("INFO: solution: %s\n", solution)

	moves, err := c.Apply(steps, *verbose)
	if err != nil {
		fmt.Printf("ERROR: Apply(%v) error: %v", steps, err)
		return
	}
	w.Write([]byte(fmt.Sprintf("OK: %s %v", solution, moves)))
	if !*verbose {
		c.Print()
	}

	log.Printf("SUCCEEDED: move: %v", moves)
}

func main() {
	flag.Parse()

	fmt.Println("Please set colors of each pieces on each face.")
	fmt.Println("Colors are: White Red Green Blue Yellow Orange, or w r g b y o.")
	fmt.Println("(input 9 whitespace-separated colors for each face):")

	if *debug {
		*verbose = true
	}

	http.HandleFunc("/cube", httpCube)

	log.Printf("Starting http server on port %d", *port)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}
