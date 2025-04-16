package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"shogunc/cmd/generate"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := scanner.Bytes()
		switch string(input) {
		case "generate":
			generator := generate.NewGenerator()
			if err := generator.Execute(); err != nil {
				log.Fatalf("Generating failed: %v", err)
			}
			// fmt.Printf("queries: %s\n", generator.Queries)
			// fmt.Printf("schema: %s\n", generator.Schema)
			// fmt.Printf("driver: %s\n", generator.Driver)
		default:
			defaultCommand()
		}
		return
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func defaultCommand() {
	fmt.Println("something...")
}
