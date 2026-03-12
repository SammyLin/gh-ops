package main

import (
	"embed"

	"github.com/SammyLin/gh-ops/cmd"
)

//go:embed web/templates/* web/static/*
var templateFS embed.FS

func main() {
	cmd.SetTemplateFS(templateFS)
	cmd.Execute()
}
