package main

import (
	"log"

	"github.com/impr0ver/metrics-service/internal/crypt"
)

func main() {
	err := crypt.GenKeys("./")
	if err != nil {
		log.Fatal(err)
	}
}
