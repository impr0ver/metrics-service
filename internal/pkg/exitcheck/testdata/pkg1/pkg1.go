package main

import (
	"fmt"
	"os"
)

func someFunc(i int) (int, error) {
	return i + i, nil
}

func Exit(number int) {
	fmt.Println("craft exit func, ", number)
}

func main() {
	// compose our preparation: analyzer must find error,
	// in comment want
	res, err := someFunc(5)
	Exit(777) // OK
	if err != nil {
		os.Exit(1) // want "call os.Exit in main.main function is not use!"
	}
	fmt.Println(res)
	os.Exit(0) // want "call os.Exit in main.main function is not use!"
}
