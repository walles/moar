package m

import "github.com/gdamore/tcell/v2"

// Page displays text in a pager.
func (p *Pager) Page() error {
	screen, e := tcell.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}
	defer func() {
		if p.DeInit {
			screen.Fini()
			return
		}
		// See: https://github.com/walles/moar/pull/39
		w, h := screen.Size()
		screen.ShowCursor(0, h - 1)
		for x := 0; x < w; x++ {
			screen.SetContent(x, h - 1, ' ', nil, tcell.StyleDefault)
		}
		screen.Show()
	}()

	p.StartPaging(screen)
	return nil
}
