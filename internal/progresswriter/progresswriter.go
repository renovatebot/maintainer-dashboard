// Package progresswriter constructs the go-pretty progress.Writer shared by
// cmd/sync and cmd/backfill.
package progresswriter

import (
	"os"
	"time"

	"github.com/jedib0t/go-pretty/v6/progress"
	"golang.org/x/term"
)

// ciUpdateFrequency is how often the writer redraws when stdout isn't a
// terminal. go-pretty's default 250ms redraw is meant for an in-place
// terminal bar: the cursor-up escapes it relies on do nothing in a plain
// log, so every redraw just appends another near-identical block. Redrawing
// far less often keeps a liveness signal in CI logs without flooding them.
const ciUpdateFrequency = 30 * time.Second

// New returns a progress.Writer rendering to stdout and starts it rendering
// in the background.
func New() progress.Writer {
	pw := progress.NewWriter()

	if !term.IsTerminal(int(os.Stdout.Fd())) {
		pw.SetUpdateFrequency(ciUpdateFrequency)
	}

	go pw.Render()

	return pw
}
