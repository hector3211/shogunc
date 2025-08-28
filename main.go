package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"shogunc/cmd/generate"
	"shogunc/utils"
)

func main() {
	// Generate test files in development mode
	if os.Getenv("DEVELOPMENT") == "true" {
		if err := utils.GenerateTestFiles(); err != nil {
			log.Printf("Warning: failed to generate test files: %v", err)
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		os.Exit(1)
	}
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		input := scanner.Bytes()
		switch string(input) {
		case "generate":
			generator := generate.NewGenerator()
			if err := generator.Execute(cwd); err != nil {
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
