package main

import (
	"github.com/koh-sh/awspagination"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(awspagination.Analyzer)
}
