package main

import (
	"fmt"
	"math/rand"
	"time"
)

func calc(size int) {
	a := make([]int, size)
	b := make([]int, size)
	st := time.Now()

	for i := 0; i < size-1; i++ {
		a[i] = rand.Intn(size)
	}

	for i := 0; i < size-1; i++ {
		b[i] = rand.Intn(size)
	}

	t := time.Now()
	elapsed := t.Sub(st)
	fmt.Printf("LUA init: %f\n", elapsed.Seconds())

	st = time.Now()
	for i := 0; i < size-1; i++ {
		if a[i] != b[i] {
			a[i] = a[i] + b[i]
		}
	}
	t = time.Now()
	elapsed = t.Sub(st)
	fmt.Printf("LUA sum: %f\n", elapsed.Seconds())
}

func main() {
	calc(5000000)
}
