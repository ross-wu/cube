// This binary is a Rubik's cube resolver.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	port = flag.Int("port", 8080, "http server port")
)

type Color int

const (
	White Color = iota
	Red
	Green
	Blue
	Yellow
	Orange
	Unknown

	FaceTop    = "top"
	FaceLeft   = "left"
	FaceFront  = "front"
	FaceRight  = "right"
	FaceBack   = "back"
	FaceBottom = "bottom"
)

var (
	faceNames = []string{FaceTop, FaceLeft, FaceFront, FaceRight, FaceBack, FaceBottom}
	colors    = map[Color]string{
		White:   "\033[37m",
		Red:     "\033[31m",
		Green:   "\033[32m",
		Blue:    "\033[34m",
		Yellow:  "\033[33m",
		Orange:  "\033[1;31m",
		Unknown: "\033[30m",
	}
)

type Face struct {
	Pieces [9]Color
}

type Cube struct {
	faces map[string]*Face
}

func NewCube() *Cube {
	return &Cube{faces: map[string]*Face{}}
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

// Flip flips the cube bottom->front->top->back.
func (c *Cube) Flip() {
	// Rotate bottom, front, top and back.
	last := FaceTop
	saved := c.faces[FaceTop]
	for _, name := range []string{FaceFront, FaceBottom, FaceBottom, FaceBack} {
		c.faces[last] = c.faces[name]
		last = name
	}
	c.faces[FaceBack] = saved

	rotateClock(c.faces[FaceRight])
	rotateCounterclock(c.faces[FaceLeft])
}

// D performs a "down" movement.
func (c *Cube) D() {
}

// DD performs "down" movement twice.
func (c *Cube) DD() {
}

// D_ performs a counterclockwise "down" movement.
func (c *Cube) D_() {

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
		printRow(c.faces[FaceTop], row)
		fmt.Printf("│\n")
	}

	fmt.Printf("\t┌────────┼────────┼────────┬────────┐\n")

	// Print left, front, right and back faces
	for row := 0; row < 3; row++ {
		fmt.Printf("\t")
		printRow(c.faces[FaceLeft], row)
		printRow(c.faces[FaceFront], row)
		printRow(c.faces[FaceRight], row)
		printRow(c.faces[FaceBack], row)
		fmt.Printf("│\n")
	}

	fmt.Printf("\t└────────┼────────┼────────┴────────┘\n")

	// Print bottom face.
	for row := 0; row < 3; row++ {
		fmt.Printf("\t%s", indent)
		printRow(c.faces[FaceBottom], row)
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

	fmt.Println("------------------------------------------------------")
	fmt.Printf("OUTPUT:\n")
	c.Flip()
	c.Print()
}
