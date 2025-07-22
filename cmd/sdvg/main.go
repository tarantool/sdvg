package main

import (
	"github.com/tarantool/sdvg/internal/generator/app"
)

var (
	version = "dev"
)

func main() {
	application := app.NewApp(version)
	application.Run()
}
