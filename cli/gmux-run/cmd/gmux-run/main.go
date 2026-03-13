package main

import (
	"flag"
	"fmt"
)

func main() {
	adapter := flag.String("adapter", "pi", "session adapter kind")
	flag.Parse()

	fmt.Printf("gmux-run scaffold (adapter=%s)\n", *adapter)
	fmt.Printf("TODO: launch command, integrate abduco, write metadata\n")
}
