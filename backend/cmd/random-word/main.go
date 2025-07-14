package main

import (
	"fmt"
	"log"

	"github.com/davidscholberg/go-urbandict"
)

func main() {
	definition, err := urbandict.Random()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s: %s\n", definition.Word, definition.Definition)
}
