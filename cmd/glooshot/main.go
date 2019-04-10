package main

import (
	"log"

	"github.com/solo-io/glooshot/pkg/setup"
)

func main() {
	log.Fatal(setup.Run())

}
