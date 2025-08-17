/*
Copyright © 2024 Ganeshdip Dumbare <ganeshdip.dumbare@gmail.com>
*/
package main

import (
	"fmt"
	"log"
	"os"

	"deblock/cmd"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered: %v", r)
			os.Exit(1)
		}
	}()

	if len(os.Args) < 2 {
		fmt.Println("Usage: deblock <command>")
		fmt.Println("Available commands:")
		fmt.Println("  rest - Start the REST API server")
		os.Exit(1)
	}

	log.Printf("Executing command: %v", os.Args)
	cmd.Execute()
}
