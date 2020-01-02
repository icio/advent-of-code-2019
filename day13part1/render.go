// Program day13part1 to be used as:
//
//     go run ./day9part1/ ./day13part1/input | go run ./day13part1/
package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	type coord struct{ x, y int }

	m := make(map[coord]int)
	for {
		var c coord
		var t int
		n, err := fmt.Fscan(os.Stdin, &c.x, &c.y, &t)
		if err == io.EOF {
			break
		} else if err != nil {
			println(err.Error())
			os.Exit(1)
		}
		if n != 3 {
			println("Expected 3 variables on input but got", n)
			os.Exit(1)
		}
		m[c] = t
	}

	var blocks int64
	for _, t := range m {
		if t == 2 {
			blocks++
		}
	}
	fmt.Println(blocks)
}
