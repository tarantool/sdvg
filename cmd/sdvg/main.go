package main

import (
	"sdvg/internal/generator/app"
)

var (
	version string
)

func main() {
	application := app.NewApp(version)
	application.Run()
}
