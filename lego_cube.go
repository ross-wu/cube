// This binary is the client-end of the lego rubik's cube solver.
// It runs on LEGO EV3 and connects to solver server.
//
// corner: eye pos=-520
//    turn pos=[130, 400, 670, 940]
// edge: eye pos=-560
//   turn pos=0, 270, 540, 810
// mid: eye pos=-710
//
// Colors: (RGB)
//   w=195/236/237
//   g=24/88/130, 36/155/182, 32/109/169, 25/79/111
//   b=27/91/134, 25/91/133
//   o=137/220/309, 87/133/210, 75/138, 236
//   r=107/40/26, 72/38/27,
//   y=73/115/145, 192/235/334
//
// Example input:
//  "bwwbyryyr wrrrgyyoo ggboobygy wygwbyoro roobrgwwb bogbwwrgg"
//   --------- --------- --------- --------- --------- ---------
//      ^         ^        ^          ^         ^         ^
//      |         |        |          |         |         |
//     Up        Left    Front      Right      Back      Down
//
// Usage:
// $ ./lego_cube --server=169.254.60.8 \
//    --input='wrygoroog bybrgooor ybygwybww rbgrbyrwb owrbygwwg yggyrbwoo'
//
package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/ev3go/ev3dev"
)

var (
	serverAddr      = flag.String("server", "", "Cube-solver server address and port.")
	portOfFlipMotor = flag.String("flip_motor", "A", "port of the flip motor, can be A, B, C, D.")
	portOfTurnMotor = flag.String("turn_motor", "B", "port of the turn motor, can be A, B, C, D.")
	portOfEyeMotor  = flag.String("eye_motor", "C", "port of the eye motor which controls the color sensor, can be A, B, C, D.")
	eyeSensorPort   = flag.String("eye_sensor", "4", "port of the eye sensor (color sensor), can be 1, 2, 3, 4")
	speedOfFlip     = flag.Int("flip_speed", 300, "flip motor speed")
	speedOfTurn     = flag.Int("turn_speed", 300, "turn motor speed")
	input           = flag.String("input", "", "eg: 'gborrwyyw wwgobbowb ogrywyygr bggogbygo worrywyyw rogborbrb'")
	test            = flag.Bool("test", false, "test")
	debug           = flag.Bool("debug", false, "debug mode")
)

type MotorDriver string

const (
	LargeMotor  MotorDriver = "lego-ev3-l-motor"
	MediumMotor MotorDriver = "lego-ev3-m-motor"
)

var (
	defaultTimeout                 = 1500 * time.Millisecond
	flipMotor, turnMotor, eyeMotor *ev3dev.TachoMotor
	curTurn                        = 0
	moves                          map[string]func()
	faces                          map[string]string
	re                             = regexp.MustCompile(`^OK: .*\[(.*)\]$`)
)

func tachoMotorOrDie(port string, driver MotorDriver) *ev3dev.TachoMotor {
	ev3port := "ev3-ports:out" + port
	motor, err := ev3dev.TachoMotorFor(ev3port, string(driver))
	if err != nil {
		log.Printf("ERROR: TachoMotorFor(%q) error: %v", ev3port, err)
		os.Exit(255)
	}
	return motor
}

func waitPosition(motor *ev3dev.TachoMotor, pos int, timeout time.Duration) bool {
	cur, err := motor.Position()
	if err != nil {
		log.Printf("ERROR: can't get initial position: %v", err)
		return false
	}
	fmt.Printf("target=%d cur=%d", pos, cur)
	n := timeout.Microseconds()
	for n > 0 && math.Abs(float64(pos-cur)) > 5.0 {
		fmt.Printf("\rtarget=%d cur=%d   ", pos, cur)
		time.Sleep(50 * time.Millisecond)
		n -= (50 * time.Millisecond).Milliseconds()
		if cur, err = motor.Position(); err != nil {
			log.Printf("ERROR: can't get current position: %v", err)
		}
	}
	if math.Abs(float64(pos-cur)) <= 5.0 {
		fmt.Printf("ok\n")
		return true
	}
	fmt.Printf("timeout\n")
	return false
}

func connectMotors() {
	flipMotor = tachoMotorOrDie(*portOfFlipMotor, LargeMotor)
	flipMotor.Command("reset")
	flipMotor.SetSpeedSetpoint(*speedOfFlip)
	flipMotor.SetStopAction("hold")

	turnMotor = tachoMotorOrDie(*portOfTurnMotor, LargeMotor)
	turnMotor.Command("reset")
	turnMotor.SetSpeedSetpoint(*speedOfTurn)
	turnMotor.SetStopAction("hold")

	eyeMotor = tachoMotorOrDie(*portOfEyeMotor, MediumMotor)
	eyeMotor.Command("reset")
	// Set eye moto to init position.
	eyeMotor.SetSpeedSetpoint(200)
	eyeMotor.SetPositionSetpoint(1000)
	eyeMotor.Command("run-to-abs-pos")
	cur, err := eyeMotor.Position()
	if err != nil {
		log.Printf("ERROR: can't get initial position of the eye motor: %v", err)
		os.Exit(255)
	}
	cnt := 0
	for i := 0; i < 30 && cur < 990; i++ {
		fmt.Printf("\rtarget=1000 cur=%d   ", cur)
		time.Sleep(50 * time.Millisecond)
		pos, err := eyeMotor.Position()
		if err != nil {
			log.Printf("ERROR: can't get current position: %v", err)
			continue
		}
		if pos == cur {
			cnt++
			if cnt >= 3 {
				fmt.Printf("Eye motor reset ok.\n")
				break
			}
			continue
		}
		cur = pos
		cnt = 0
	}
	eyeMotor.Command("stop")
	eyeMotor.Command("reset")
	eyeMotor.SetSpeedSetpoint(200)
	eyeMotor.SetStopAction("hold")
}

func resetMotors() {
	if flipMotor != nil {
		flipMotor.Command("reset")
	}
	if turnMotor != nil {
		turnMotor.Command("reset")
	}
	if eyeMotor != nil {
		eyeMotor.Command("reset")
	}
}

func flip() {
	pos := 220
	flipMotor.SetPositionSetpoint(pos)
	flipMotor.Command("run-to-abs-pos")
	waitPosition(flipMotor, pos, defaultTimeout)
	flipMotor.Command("stop")

	time.Sleep(200 * time.Millisecond)

	flipMotor.SetPositionSetpoint(5)
	flipMotor.Command("run-to-abs-pos")
	waitPosition(flipMotor, 5, defaultTimeout)
	flipMotor.Command("stop")
}

func turn(n int) {
	newPos := curTurn + n*270
	turnMotor.SetPositionSetpoint(newPos)
	turnMotor.Command("run-to-abs-pos")
	waitPosition(turnMotor, newPos, defaultTimeout)
	curTurn = newPos
}

func holdCube() {
	const pos = 110
	flipMotor.SetPositionSetpoint(pos)
	flipMotor.Command("run-to-abs-pos")
	waitPosition(flipMotor, pos, defaultTimeout)
	flipMotor.Command("stop")
}

func releaseCube() {
	const pos = 5
	flipMotor.SetPositionSetpoint(pos)
	flipMotor.Command("run-to-abs-pos")
	waitPosition(flipMotor, pos, defaultTimeout)
	flipMotor.Command("stop")
}

func initMoves() {
	moves = map[string]func(){
		"flip": func() {
			flip()
		},
		"turn": func() {
			turn(1)
		},
		"turn2": func() {
			turn(2)
		},
		"turn'": func() {
			turn(-1)
		},
		"D": func() {
			holdCube()
			turn(-1)
			releaseCube()
		},
		"D2": func() {
			holdCube()
			turn(-2)
			releaseCube()
		},
		"D'": func() {
			holdCube()
			turn(1)
			releaseCube()
		},
	}
}

func parseInput(input string) {
	faces = map[string]string{}
	arr := strings.Split(input, " ")
	if len(arr) != 6 {
		fmt.Printf("ERROR: wrong input: len(arr)=%d: %q", len(arr), input)
		os.Exit(255)
	}
	faces["U"] = arr[0]
	faces["L"] = arr[1]
	faces["F"] = arr[2]
	faces["R"] = arr[3]
	faces["B"] = arr[4]
	faces["D"] = arr[5]
}

func sendRequest() ([]string, error) {
	req := fmt.Sprintf("http://%s/cube?U=%s&L=%s&F=%s&R=%s&B=%s&D=%s",
		*serverAddr, faces["U"], faces["L"], faces["F"], faces["R"], faces["B"], faces["D"])
	log.Printf("GET %s", req)
	resp, err := http.Get(req)
	if err != nil {
		log.Printf("ERROR: GET error: %v", err)
		return nil, err
	}

	fmt.Printf("Response:\nStatus: %d: %s\n\n", resp.StatusCode, resp.Status)
	scanner := bufio.NewScanner(resp.Body)
	var body string
	for i := 0; scanner.Scan() && i < 5; i++ {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 2 {
			body = matches[1]
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("ERROR: read body error: %v", err)
	}
	if body == "" {
		log.Printf("ERROR: Can't get solution from server.\n")
		return nil, fmt.Errorf("solution not found")
	}

	steps := strings.Split(body, " ")
	log.Printf("stpes: %+v\n", steps)

	return steps, nil
}

func Move(m string) {
	op, ok := moves[m]
	if !ok {
		log.Printf("ERROR: unknown move %q", m)
		return
	}
	op()
}

func solve(steps []string) {
	n := len(steps)
	fmt.Printf("TOTAL STEPS=%d:\n", n)
	var reader *bufio.Reader
	if *debug {
		reader = bufio.NewReader(os.Stdin)
	}
	for i, m := range steps {
		fmt.Printf(">>> STEP %d/%d: %s\n", i, n, m)
		if reader != nil {
			reader.ReadString('\n')
		}
		Move(m)
	}

	turn(8)
}

func main() {
	flag.Parse()

	connectMotors()
	initMoves()

	if *test {
		fmt.Printf("Flip two times.\n")
		flip()
		flip()

		fmt.Printf("Action: turn reverseTurn turn2\n")
		time.Sleep(time.Second)
		turn(1)
		turn(-1)
		turn(2)

		fmt.Printf("Moves: D D' D2\n")
		time.Sleep(time.Second)
		Move("D")
		Move("D'")
		Move("D2")
		fmt.Printf("\nDONE TEST\n")

		resetMotors()
		os.Exit(0)
	}

	if *input == "" {
		fmt.Printf("ERROR: --input is empty!")
		os.Exit(1)
	}
	if *input != "" {
		parseInput(*input)
	}

	if *serverAddr == "" {
		fmt.Println("ERROR: --server must be set.")
		os.Exit(1)
	}

	steps, err := sendRequest()
	if err != nil {
		fmt.Printf("ERROR: request server error: %v", err)
		os.Exit(255)
	}
	fmt.Printf("Response: steps=%d: %v\n", len(steps), steps)
	solve(steps)

	resetMotors()
	fmt.Printf("\nDONE\n")
}
