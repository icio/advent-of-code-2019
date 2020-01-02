package main

import (
	"fmt"
	"io"
	"os"
)

func main() {
	var modules, fuel int64
	for {
		var mass int64
		_, err := fmt.Fscanln(os.Stdin, &mass)
		if err == io.EOF {
			break
		}
		modules++
		fuel += fuelForMass(mass)
	}
	fmt.Printf("%d modules requiring %d fuel.\n", modules, fuel)
}

func fuelForMass(mass int64) int64 {
	return mass/3 - 2
}
