package m

import "github.com/gdamore/tcell/v2"

// Page displays text in a pager.
//
// The reader parameter can be constructed using one of:
// * NewReaderFromFilename()
// * NewReaderFromText()
// * NewReaderFromStream()
//
// Or your could roll your own Reader based on the source code for any of those
// constructors.
func Page(reader *Reader) error {
	screen, e := tcell.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}
	defer screen.Fini()

	NewPager(reader).StartPaging(screen)
	return nil
}
