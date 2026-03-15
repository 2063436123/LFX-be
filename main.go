package main

import (
	"log"

	"lfx-be/internal/server"
)

func main() {
	app := server.New()
	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
