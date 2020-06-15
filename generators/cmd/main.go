package main

import (
	"fmt"
	"log"
	"os"

	"generators/pkg/parser"
)

func main() {
	fmt.Println("Yaml Parser Demo:")
	products, err := parser.Parse(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(products)
}
