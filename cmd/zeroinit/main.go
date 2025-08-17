package main

import (
	"github.com/sivchari/zeroinit"
	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(zeroinit.Analyzer) }
