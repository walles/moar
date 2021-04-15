package m

import "github.com/walles/moar/twin"

// Page displays text in a pager.
func (p *Pager) Page() error {
	screen, e := twin.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}
	defer func() {
		if p.DeInit {
			screen.Close()
			return
		}

		// See: https://github.com/walles/moar/pull/39
		// FIXME: Consider moving this logic into the twin package.
		w, h := screen.Size()
		screen.ShowCursorAt(0, h-1)
		for x := 0; x < w; x++ {
			screen.SetCell(x, h-1, twin.NewCell(' ', twin.StyleDefault))
		}
		screen.Show()
	}()

	p.StartPaging(screen)
	return nil
}
