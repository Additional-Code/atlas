package main

import (
	"go.uber.org/fx"

	"github.com/Additional-Code/atlas/internal/app"
)

func main() {
	fx.New(app.Module).Run()
}
