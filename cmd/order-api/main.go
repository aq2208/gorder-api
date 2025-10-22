package main

import (
	"log"
	"os"

	"github.com/aq2208/gorder-api/cmd/order-api/app"
	"github.com/aq2208/gorder-api/configs"
)

func main() {
	env := os.Getenv("APP_ENV") // dev | staging | prod
	if env == "" {
		env = "dev"
	}

	cfg, err := configs.Load("configs", env)
	if err != nil {
		log.Fatal(err)
	}

	app, cleanup, err := app.InitWithConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	log.Printf("order-api (%s) listening on %s", env, cfg.App.HTTPAddr)
	if err := app.Router.Run(cfg.App.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
