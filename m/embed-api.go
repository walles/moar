package m

import "github.com/walles/moar/twin"

// If true, the Page function will clear the screen on return. If false, the
// Page function will clear the last line, and show the cursor.
var DeInit = true

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
	screen, e := twin.NewScreen()
	if e != nil {
		// Screen setup failed
		return e
	}
	defer func() {
		if DeInit {
			screen.Close()
		} else {
			// See: https://github.com/walles/moar/pull/39
			// FIXME: Consider moving this logic into the twin package.
			w, h := screen.Size()
			screen.ShowCursorAt(0, h-1)
			for x := 0; x < w; x++ {
				screen.SetCell(x, h-1, twin.NewCell(' ', twin.StyleDefault))
			}
			screen.Show()
		}
	}()

	NewPager(reader).StartPaging(screen)
	return nil
}
