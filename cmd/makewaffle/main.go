package main

import (
	"os"

	"github.com/dalemusser/waffle/internal/wafflegen"
)

func main() {
	os.Exit(wafflegen.Run("makewaffle", os.Args[1:]))
}
