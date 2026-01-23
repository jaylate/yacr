package main

import (
	"fmt"
	"os"
)

func main() {
	usageMsg := "Usage: " + os.Args[0] + " run <command> <arguments>"
	if len(os.Args) > 1 {
		if os.Args[1] == "run" && len(os.Args) > 2 {
			command := os.Args[2]
			args := os.Args[3:]
			executeInNamespace(command, args...)
		} else {
			fmt.Println("Incorrect usage")
			fmt.Println(usageMsg)
		}
	} else {
		fmt.Println(usageMsg)
	}
}
