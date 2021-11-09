package m

import (
	"github.com/walles/moar/twin"
)

// Page displays text in a pager.
func (p *Pager) Page() error {
	screen, e := twin.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}

	p.StartPaging(screen)
	screen.Close()
	if p.DeInit {
		return nil
	}

	return p.ReprintAfterExit()
}
