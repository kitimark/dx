package main

import (
	"github.com/fatih/color"
	"nmyk.io/cowsay"
)

func main() {
	cowsay.Cowsay("hello world")
	color.Cyan("wow, it prints a color")
}
