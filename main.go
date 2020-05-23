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
	WrongColor

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
		White:  "\033[37m",
		Red:    "\033[31m",
		Green:  "\033[32m",
		Blue:   "\033[34m",
		Yellow: "\033[33m",
		Orange: "\033[1;31m",
	}
)

type Face struct {
	Pieces [9]Color
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

func printCube(faces map[string]*Face) {
	const indent = "         "

	fmt.Printf("\t         ┌────────┐\n")

	// Print top face.
	for row := 0; row < 3; row++ {
		fmt.Printf("\t%s", indent)
		printRow(faces[FaceTop], row)
		fmt.Printf("│\n")
	}

	fmt.Printf("\t┌────────┼────────┼────────┬────────┐\n")

	// Print left, front, right and back faces
	for row := 0; row < 3; row++ {
		fmt.Printf("\t")
		printRow(faces[FaceLeft], row)
		printRow(faces[FaceFront], row)
		printRow(faces[FaceRight], row)
		printRow(faces[FaceBack], row)
		fmt.Printf("│\n")
	}

	fmt.Printf("\t└────────┼────────┼────────┴────────┘\n")

	// Print bottom face.
	for row := 0; row < 3; row++ {
		fmt.Printf("\t%s", indent)
		printRow(faces[FaceBottom], row)
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
	return WrongColor
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
			if face.Pieces[i] == WrongColor {
				fmt.Printf("WARNING: wrong color %s for piece %d, skip.", s, i)
			}
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

	faces := map[string]*Face{}
	reader := bufio.NewReader(os.Stdin)
	for _, faceName := range faceNames {
		faces[faceName] = readFace(faceName, reader)
	}

	fmt.Printf("\n\nInput:\n")
	printCube(faces)

}
