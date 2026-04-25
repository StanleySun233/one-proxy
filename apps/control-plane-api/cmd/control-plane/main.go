package main

import (
	"log"

	"github.com/StanleySun233/python-proxy/apps/control-plane-api/internal/app"
)

func main() {
	application := app.New()
	if err := application.Run(); err != nil {
		log.Fatal(err)
	}
}
