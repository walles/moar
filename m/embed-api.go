package m

import (
	"fmt"
	"github.com/walles/moar/twin"
)

// Page displays text in a pager.
func (p *Pager) Page() error {
	screen, e := twin.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}
	defer func() {
		screen.Close()
		if p.DeInit { return }

		// FIXME: Consider moving this logic into the twin package.
		_, height := p.screen.Size()
		lines := p.reader.GetLines(p.firstLineOneBased, height-1).lines
		for _, line := range lines {
			fmt.Println(*line.raw)
		}
	}()

	p.StartPaging(screen)
	return nil
}
