package main

import (
	"errors"
	"fmt"
	"io"
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
	intcode := make([]int, len(codeop))
	for i, op := range codeop {
		n, err := strconv.Atoi(op)
		if err != nil {
			log.Fatalf("Failed to parse int %q at position %d.", op, i)
		}
		intcode[i] = n
	}

	if err := solve(intcode); err != nil {
		log.Fatal(err)
	}
}

func solve(intcode []int) error {
	// Space for each instance of the program that we execute, to be reset
	// before each execution.
	inst := make([]int, len(intcode))
	max := 0

	// Enumerate the phase setting under each program.
	for sa := 0; sa < 5; sa++ {
		ra, err := amplify(intcode, inst, sa, 0)
		if err != nil {
			return err
		}

		for sb := 0; sb < 5; sb++ {
			if sb == sa {
				continue
			}

			rb, err := amplify(intcode, inst, sb, ra)
			if err != nil {
				return err
			}

			for sc := 0; sc < 5; sc++ {
				if sc == sa || sc == sb {
					continue
				}

				rc, err := amplify(intcode, inst, sc, rb)
				if err != nil {
					return err
				}

				for sd := 0; sd < 5; sd++ {
					if sd == sa || sd == sb || sd == sc {
						continue
					}

					rd, err := amplify(intcode, inst, sd, rc)
					if err != nil {
						return err
					}

					for se := 0; se < 5; se++ {
						if se == sa || se == sb || se == sc || se == sd {
							continue
						}

						re, err := amplify(intcode, inst, se, rd)
						if err != nil {
							return err
						}
						if re >= max {
							max = re
							fmt.Printf("%d,%d,%d,%d,%d => %d\n", sa, sb, sc, sd, se, re)
						}
					}
				}
			}
		}
	}

	fmt.Println(max)
	return nil
}

func amplify(intcode, inst []int, phase, inp int) (int, error) {
	ia := make(chan int, 2)
	ia <- phase
	ia <- inp
	close(ia)
	oa := make(chan int, 1)
	copy(inst, intcode)
	err := exec(chanio{ia, oa}, inst)
	close(oa)
	return <-oa, err
}

type execio interface {
	Input() (int, error)
	Output(int) error
}

type stdio struct{}

func (stdio) Input() (int, error) {
	var v int
	for {
		fmt.Printf("Enter integer: ")
		n, err := fmt.Fscanln(os.Stdin, &v)
		if err == io.EOF {
			return 0, err
		} else if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		} else if n != 1 {
			fmt.Fprintln(os.Stderr, "Please provide one integer.")
			continue
		}
		return v, nil
	}
}

func (stdio) Output(n int) error {
	_, err := fmt.Println(n)
	return err
}

type chanio struct {
	in  <-chan int
	out chan<- int
}

func (p chanio) Input() (int, error) {
	n, open := <-p.in
	if !open {
		return 0, fmt.Errorf("input closed")
	}
	return n, nil
}

func (p chanio) Output(n int) error {
	p.out <- n
	return nil
}

func exec(io execio, intcode []int) error {
	opn := 0
	for opn < len(intcode) {
		op := intcode[opn]
		switch op % 100 {
		case 99:
			// Return.
			fmt.Fprintf(os.Stderr, "% 4d: ret(99)\n", opn)
			return nil
		case 1:
			// Add.
			a, b, ans, err := readParamParamAddr(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("add(1): %s", err)
			}
			vc := a.v + b.v
			intcode[ans] = vc
			fmt.Fprintf(os.Stderr, "% 4d: add(1): %s + %s = %d -> *%d\n", opn, a, b, vc, ans)
			opn += 4
		case 2:
			// Multiply.
			a, b, ans, err := readParamParamAddr(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("mul(2): %s", err)
			}
			vc := a.v * b.v
			intcode[ans] = vc
			fmt.Fprintf(os.Stderr, "% 4d: mul(2): %s + %s = %d -> *%d\n", opn, a, b, vc, ans)
			opn += 4
		case 3:
			// Input.
			dst, err := readAddr(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("inp(3): %s", err)
			}
			v, err := io.Input()
			if err != nil {
				return fmt.Errorf("inp(3): reading input: %w", err)
			}
			intcode[dst] = v
			fmt.Fprintf(os.Stderr, "% 4d: inp(3): %d -> *%d\n", opn, v, dst)
			opn += 2
		case 4:
			// Output.
			src, err := readParam(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("out(4): %s", err)
			}
			fmt.Fprintf(os.Stderr, "% 4d: out(4): %s\n", opn, src)
			err = io.Output(src.v)
			if err != nil {
				return fmt.Errorf("out(4): writing output: %w", err)
			}
			opn += 2
		case 5:
			// Jump-if-True.
			cond, jump, err := readParamParam(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("jtr(5): %s", err)
			}
			if cond.v == 0 {
				fmt.Fprintf(os.Stderr, "% 4d: jtr(5): %s == 0\n", opn, cond)
				opn += 3
			} else {
				fmt.Fprintf(os.Stderr, "% 4d: jtr(5): %s != 0 => %s\n", opn, cond, jump)
				opn = jump.v
			}
		case 6:
			// Jump-if-False.
			cond, jump, err := readParamParam(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("jfa(6): %s", err)
			}
			if cond.v != 0 {
				fmt.Fprintf(os.Stderr, "% 4d: jfa(6): %s != 0\n", opn, cond)
				opn += 3
			} else {
				fmt.Fprintf(os.Stderr, "% 4d: jfa(6): %s == 0 => %s\n", opn, cond, jump)
				opn = jump.v
			}
		case 7:
			// Less than.
			a, b, ans, err := readParamParamAddr(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("les(7): %s", err)
			}
			if a.v < b.v {
				intcode[ans] = 1
			} else {
				intcode[ans] = 0
			}
			fmt.Fprintf(os.Stderr, "% 4d: les(7): %s < %s = %d -> *%d\n", opn, a, b, intcode[ans], ans)
			opn += 4
		case 8:
			// Equals.
			a, b, ans, err := readParamParamAddr(intcode, opn, 1)
			if err != nil {
				return fmt.Errorf("equ(8): %s", err)
			}
			if a.v == b.v {
				intcode[ans] = 1
			} else {
				intcode[ans] = 0
			}
			fmt.Fprintf(os.Stderr, "% 4d: equ(8): %s == %s = %d -> *%d\n", opn, a, b, intcode[ans], ans)
			opn += 4
		default:
			return fmt.Errorf("intcode: unrecognised op %d at position %d", op%100, opn)
		}
	}
	return errors.New("intcode: no operation")
}

type param struct {
	p int
	v int
}

func (p param) String() string {
	if p.p < 0 {
		return strconv.Itoa(p.v)
	}
	return "(*" + strconv.Itoa(p.p) + " -> " + strconv.Itoa(p.v) + ")"
}

func readLiteralFlag(intcode []int, opn int, n int) bool {
	// fmt.Fprintf(os.Stderr, "% 4d: lit = %05d, exp10(%d+1) = %d, ...%%10 = %d\n", opn, intcode[opn], n, exp10(n+1), (intcode[opn]/exp10(n+1))%10)
	return (intcode[opn]/exp10(n+1))%10 != 0
}

func readParamParamAddr(intcode []int, opn int, n int) (a param, b param, addr int, err error) {
	a, b, err = readParamParam(intcode, opn, n)
	if err != nil {
		return
	}
	addr, err = readAddr(intcode, opn, n+2)
	return
}

func readParamParam(intcode []int, opn int, n int) (a param, b param, err error) {
	a, err = readParam(intcode, opn, n)
	if err != nil {
		return
	}
	b, err = readParam(intcode, opn, n+1)
	return
}

func readParam(intcode []int, opn int, n int) (param, error) {
	// Check we have the intcode we want.
	if opn+n >= len(intcode) {
		return param{}, fmt.Errorf("wanted %d parameters but got %d at position %d", n, len(intcode)-opn-1, opn)
	}

	// Determine whether the parameter is a literal or pointer.
	if readLiteralFlag(intcode, opn, n) {
		return param{p: -1, v: intcode[opn+n]}, nil
	}

	// Resolve pointers.
	p := intcode[opn+n]
	if p >= len(intcode) {
		return param{}, fmt.Errorf("invalid address %d at position %d", p, opn+n)
	}
	return param{p: p, v: intcode[p]}, nil
}

func readAddr(intcode []int, opn int, n int) (int, error) {
	if readLiteralFlag(intcode, opn, n) {
		return -1, fmt.Errorf("wanted pointer but received literal at position %d (op=%05d, )", opn+n, intcode[opn])
	}
	p := intcode[opn+n]
	if p >= len(intcode) {
		return -1, fmt.Errorf("invalid address %d at position %d", p, opn+n)
	}
	return p, nil
}

func exp10(n int) int {
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
	v := 1
	for ; n > 0; n-- {
		v *= 10
	}
	return v
}
