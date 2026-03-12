package main

import (
	"embed"
	"flag"
	"log"

	"github.com/SammyLin/gh-ops/cmd"
)

//go:embed web/templates/* web/static/*
var templateFS embed.FS

func main() {
	configPath := flag.String("config", "config.yaml", "path to config file")
	flag.Parse()

	if err := cmd.Run(*configPath, templateFS); err != nil {
		log.Fatal(err)
	}
}
