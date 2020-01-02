package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

func main() {
	// Read the code.
	var code string
	_, err := fmt.Fscanln(os.Stdin, &code)
	if err != nil {
		log.Fatalln(err)
	}

	// Parse the code into operators.
	codeop := strings.Split(code, ",")
	intcode := make([]int, len(codeop))
	for i, op := range codeop {
		n, err := strconv.Atoi(op)
		if err != nil {
			log.Fatalf("Failed to parse int %q at position %d.", op, i)
		}
		intcode[i] = n
	}

	fmt.Println(exec(intcode))
}

func exec(intcode []int) (int, error) {
	opn := 0
	for opn < len(intcode) {
		op := intcode[opn]
		switch op {
		case 99:
			// Return.
			fmt.Fprintf(os.Stderr, "% 4d: ret *0 = %d\n", opn, intcode[0])
			return intcode[0], nil
		case 1:
			// Add.
			a, b, c, err := read3refs(intcode, opn+1)
			if err != nil {
				return 0, fmt.Errorf("add(1): %w", err)
			}
			va, vb := intcode[a], intcode[b]
			vc := va + vb
			intcode[c] = vc
			fmt.Fprintf(os.Stderr, "% 4d: add *%d *%d = %d + %d = %d -> *%d\n", opn, a, b, va, vb, vc, c)
			opn += 4
		case 2:
			// Multiply.
			a, b, c, err := read3refs(intcode, opn+1)
			if err != nil {
				return 0, fmt.Errorf("mul(2): %w", err)
			}
			va, vb := intcode[a], intcode[b]
			vc := va * vb
			intcode[c] = vc
			fmt.Fprintf(os.Stderr, "% 4d: mul *%d *%d = %d * %d = %d -> *%d\n", opn, a, b, va, vb, vc, c)
			opn += 4
		default:
			return 0, fmt.Errorf("intcode: unrecognised op %d at position %d", op, opn)
		}
	}
	return 0, errors.New("intcode: no operation")
}

func read3refs(intcode []int, opn int) (int, int, int, error) {
	if opn+3 > len(intcode) {
		return 0, 0, 0, fmt.Errorf("expected 3 arguments but received %d", len(intcode)-opn-1)
	}

	a, b, c := intcode[opn], intcode[opn+1], intcode[opn+2]
	m := c
	if a >= b && a >= c {
		m = a
	} else if b >= a && b >= c {
		m = b
	}
	if m >= len(intcode) {
		return 0, 0, 0, fmt.Errorf("register %d does not exist", m)
	}
	return a, b, c, nil
}
