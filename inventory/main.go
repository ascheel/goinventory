package main

import (
	"fmt"
	"log"
	_ "embed"

	"github.com/ascheel/goinventory/inventory/inventoryengine"
)

// Populated through the make command during the build phase.
var Version string

func printVersion() {
	fmt.Println("Version: ", Version)
}

func main() {
	printVersion()

	i := inventoryengine.NewInventory()
	err := i.Roll()

	if err != nil {
		log.Fatalln(err)
	}
}
