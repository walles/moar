package m

import "github.com/gdamore/tcell/v2"

// PageString displays a multi-line text in a pager.
//
// name - Will be displayed in the bottom left corner of the pager window
//
// text - This is the (potentially long) multi line text that will be displayed
func PageString(name string, text string) error {
	reader := NewReaderFromText(name, text)

	screen, e := tcell.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}
	defer screen.Fini()

	NewPager(reader).StartPaging(screen)
	return nil
}
