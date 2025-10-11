package main

import (
	"log"
	"os"

	"github.com/aq2208/gorder-api/internal/bootstrap"
	"github.com/aq2208/gorder-api/internal/config"
)

func main() {
	env := os.Getenv("APP_ENV") // dev | staging | prod
	if env == "" {
		env = "dev"
	}

	cfg, err := config.Load("configs", env)
	if err != nil {
		log.Fatal(err)
	}

	app, cleanup, err := bootstrap.InitWithConfig(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer cleanup()

	log.Printf("order-api (%s) listening on %s", env, cfg.App.HTTPAddr)
	if err := app.Router.Run(cfg.App.HTTPAddr); err != nil {
		log.Fatal(err)
	}
}
