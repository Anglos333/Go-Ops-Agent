package main

import (
	"log"

	"go-ops-agent/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
