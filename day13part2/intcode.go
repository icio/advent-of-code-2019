package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Read the code.
	code, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalln(err)
	}

	// Parse the code into operators.
	codeop := strings.Split(strings.TrimSpace(string(code)), ",")
	intcode := make([]int64, len(codeop))
	for i, op := range codeop {
		n, err := strconv.ParseInt(op, 10, 64)
		if err != nil {
			log.Fatalf("Failed to parse int %q at position %d.", op, i)
		}
		intcode[i] = n
	}

	// From the puzzle instructions: setting instruction 0 to 2 provides
	// repeated play of the game.
	intcode[0] = 2

	player := newPaddleAI()
	err = exec(&prog{
		io:  player,
		mem: intcode,
	})
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(player.score)
}

type prog struct {
	io interface {
		Input() (int64, error)
		Output(int64) error
	}
	mem  []int64
	base int
}

func (p *prog) get(r int) int64 {
	if r > len(p.mem) {
		return 0
	}
	return p.mem[int(r)]
}

func (p *prog) set(r int, v int64) {
	if len(p.mem) <= r {
		c := cap(p.mem)
		for c < r {
			c *= 2
		}
		m := p.mem
		p.mem = make([]int64, c)
		copy(p.mem, m)
	}
	p.mem[r] = v
}

type paddleAI struct {
	score int64
	world map[coord]tile

	display  [3]int64
	displayN int
}

type coord struct{ x, y int64 }
type tile int64

const (
	tileEmpty  = 0
	tileWall   = 1
	tileBlock  = 2
	tilePaddle = 3
	tileBall   = 4
)

func newPaddleAI() *paddleAI {
	return &paddleAI{
		world: make(map[coord]tile),
	}
}

// paddleAI.Input returns the integer left-right movement of the paddle with the
// joystick, where the position of the joystick is determined as:
//
// *  0: neutral position
// * -1: left position
// * +1: right position
func (p *paddleAI) Input() (int64, error) {
	// Ball/Paddle Left/right.
	bl := int64(-1)
	br := int64(-1)
	pl := int64(-1)
	pr := int64(-1)

	// Find the left and right extents of the paddle and ball.
	for pos, t := range p.world {
		switch t {
		case tileBall:
			if bl == -1 || bl > pos.x {
				bl = pos.x
			}
			if br == -1 || br < pos.x {
				br = pos.x
			}
		case tilePaddle:
			if pl == -1 || pl > pos.x {
				pl = pos.x
			}
			if pr == -1 || pr < pos.x {
				pr = pos.x
			}
		}
	}

	// Move the joystick to keep the centers aligned.
	bc := (bl + br) / 2
	pc := (pl + pr) / 2
	switch {
	case bc < pc:
		return -1, nil
	case bc > pc:
		return 1, nil
	default:
		return 0, nil
	}
}

func (p *paddleAI) Output(n int64) error {
	// Collect display.
	p.display[p.displayN] = n
	p.displayN++
	if p.displayN < 3 {
		return nil
	}
	p.displayN = 0

	// Track the output as tiles/scores.
	x, y, t := p.display[0], p.display[1], p.display[2]
	if x == -1 && y == 0 {
		p.score = t
	} else {
		p.world[coord{x, y}] = tile(t)
	}
	return nil
}

func exec(p *prog) error {
	opn := 0
	for opn < len(p.mem) {
		op := p.mem[opn]
		switch op % 100 {
		case 99:
			// Return.
			fmt.Fprintf(os.Stderr, "% 4d: ret(99)\n", opn)
			return nil
		case 1:
			// Add.
			a, b, ans, err := readParamParamAddr(p, opn, 1)
			if err != nil {
				return fmt.Errorf("add(1): %s", err)
			}
			vc := a.v + b.v
			p.set(ans, vc)
			fmt.Fprintf(os.Stderr, "% 4d: add(1): %s + %s = %d -> *%d\n", opn, a, b, vc, ans)
			opn += 4
		case 2:
			// Multiply.
			a, b, ans, err := readParamParamAddr(p, opn, 1)
			if err != nil {
				return fmt.Errorf("mul(2): %s", err)
			}
			vc := a.v * b.v
			p.set(ans, vc)
			fmt.Fprintf(os.Stderr, "% 4d: mul(2): %s + %s = %d -> *%d\n", opn, a, b, vc, ans)
			opn += 4
		case 3:
			// Input.
			dst, err := readAddr(p, opn, 1)
			if err != nil {
				return fmt.Errorf("inp(3): %s", err)
			}
			v, err := p.io.Input()
			if err != nil {
				return fmt.Errorf("inp(3): reading input: %w", err)
			}
			p.set(dst, v)
			fmt.Fprintf(os.Stderr, "% 4d: inp(3): %d -> *%d\n", opn, v, dst)
			opn += 2
		case 4:
			// Output.
			src, err := readParam(p, opn, 1)
			if err != nil {
				return fmt.Errorf("out(4): %s", err)
			}
			fmt.Fprintf(os.Stderr, "% 4d: out(4): %s\n", opn, src)
			err = p.io.Output(src.v)
			if err != nil {
				return fmt.Errorf("out(4): writing output: %w", err)
			}
			opn += 2
		case 5:
			// Jump-if-True.
			cond, jump, err := readParamParam(p, opn, 1)
			if err != nil {
				return fmt.Errorf("jtr(5): %s", err)
			}
			if cond.v != 0 {
				// True.
				fmt.Fprintf(os.Stderr, "% 4d: jtr(5): %s != 0 => %s\n", opn, cond, jump)
				opn = int(jump.v)
			} else {
				fmt.Fprintf(os.Stderr, "% 4d: jtr(5): %s == 0\n", opn, cond)
				opn += 3
			}
		case 6:
			// Jump-if-False.
			cond, jump, err := readParamParam(p, opn, 1)
			if err != nil {
				return fmt.Errorf("jfa(6): %s", err)
			}
			if cond.v == 0 {
				// False.
				fmt.Fprintf(os.Stderr, "% 4d: jfa(6): %s == 0 => %s\n", opn, cond, jump)
				opn = int(jump.v)
			} else {
				fmt.Fprintf(os.Stderr, "% 4d: jfa(6): %s != 0\n", opn, cond)
				opn += 3
			}
		case 7:
			// Less than.
			a, b, ans, err := readParamParamAddr(p, opn, 1)
			if err != nil {
				return fmt.Errorf("les(7): %s", err)
			}
			var v int64
			if a.v < b.v {
				v = 1
			}
			p.set(ans, v)
			fmt.Fprintf(os.Stderr, "% 4d: les(7): %s < %s = %d -> *%d\n", opn, a, b, v, ans)
			opn += 4
		case 8:
			// Equals.
			a, b, ans, err := readParamParamAddr(p, opn, 1)
			if err != nil {
				return fmt.Errorf("equ(8): %s", err)
			}
			var v int64
			if a.v == b.v {
				v = 1
			}
			p.set(ans, v)
			fmt.Fprintf(os.Stderr, "% 4d: equ(8): %s == %s = %d -> *%d\n", opn, a, b, v, ans)
			opn += 4
		case 9:
			// Base.
			base, err := readParam(p, opn, 1)
			if err != nil {
				return fmt.Errorf("bas(9): %s", err)
			}
			b := p.base
			p.base += int(base.v)
			fmt.Fprintf(os.Stderr, "% 4d: bas(9): %d + %s ~> %d\n", opn, b, base, p.base)
			opn += 2
		default:
			return fmt.Errorf("intcode: unrecognised op %d at position %d", op%100, opn)
		}
	}
	return errors.New("intcode: no operation")
}

type param struct {
	f int
	p int
	r int
	v int64
}

func (p param) String() string {
	switch p.f {
	case flagLit:
		return strconv.FormatInt(p.v, 10)
	case flagPos:
		return "(*" + strconv.Itoa(p.p) + " -> " + strconv.FormatInt(p.v, 10) + ")"
	case flagRel:
		return "(*" + strconv.Itoa(p.p) + "-" + strconv.Itoa(p.r) + " -> " + strconv.FormatInt(p.v, 10) + ")"
	}
	return fmt.Sprintf("%#v", p)
}

func readParamParamAddr(pr *prog, opn int, n int) (a param, b param, addr int, err error) {
	a, b, err = readParamParam(pr, opn, n)
	if err != nil {
		return
	}
	addr, err = readAddr(pr, opn, n+2)
	return
}

func readParamParam(pr *prog, opn int, n int) (a param, b param, err error) {
	a, err = readParam(pr, opn, n)
	if err != nil {
		return
	}
	b, err = readParam(pr, opn, n+1)
	return
}

func readParam(pr *prog, opn int, n int) (param, error) {
	f := readFlag(pr, opn, n)
	var p, r int
	switch f {
	case flagLit:
		return param{f: f, p: -1, v: pr.get(opn + n)}, nil
	case flagPos:
		p = int(pr.get(opn + n))
	case flagRel:
		r = int(pr.get(opn + n))
		p = pr.base + r
	}
	return param{f: f, p: p, r: r, v: pr.get(p)}, nil
}

func readAddr(pr *prog, opn int, n int) (int, error) {
	f := readFlag(pr, opn, n)
	switch f {
	case flagLit:
		return -1, fmt.Errorf("wanted pointer but literal at position %d", opn+n)
	case flagPos:
		return int(pr.get(opn + n)), nil
	case flagRel:
		return pr.base + int(pr.get(opn+n)), nil
	default:
		return -1, fmt.Errorf("unrecognised flag %d", f)
	}
}

func readFlag(pr *prog, opn int, n int) int {
	return int((pr.get(opn) / exp10(n+1)) % 10)
}

const (
	flagPos = 0 // Positional mode: the value is at the address
	flagLit = 1 // Immediate mode: the value is literal
	flagRel = 2 // Relative mode: the value is at the address relative to the root
)

func exp10(n int) int64 {
	switch n {
	case 0:
		return 1e0
	case 1:
		return 1e1
	case 2:
		return 1e2
	case 3:
		return 1e3
	case 4:
		return 1e4
	}
	v := int64(1)
	for ; n > 0; n-- {
		v *= 10
	}
	return v
}
