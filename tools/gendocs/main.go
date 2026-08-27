// Command gendocs regenerates docs/impactline.1 from Cobra command metadata.
// It is the source of truth for the man page: docs/impactline.1 is a
// committed, generated artifact, not hand-authored. Run via `make man`.
package main

import (
	"log"
	"os"

	"github.com/spf13/cobra/doc"

	"github.com/AlbertoBarrago/impactline/cmd"
)

func main() {
	outDir := "docs"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}

	header := &doc.GenManHeader{
		Title:   "IMPACTLINE",
		Section: "1",
		Source:  "Impactline",
		Manual:  "Impactline Manual",
	}

	if err := doc.GenManTree(cmd.RootCmd(), header, outDir); err != nil {
		log.Fatalf("gendocs: %v", err)
	}
}
