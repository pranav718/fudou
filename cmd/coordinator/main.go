package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("Fudou Coordinator Service starting...")
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	return nil
}
