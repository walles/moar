package internal

import (
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/styles"
	log "github.com/sirupsen/logrus"
	"github.com/walles/moor/v2/twin"
)

const defaultDarkTheme = "native"

// I decided on a light theme by doing this:
//
//	wc -l ../chroma/styles/*.xml|sort|cut -d/ -f4|grep xml|xargs -I XXX grep -Hi background ../chroma/styles/XXX
//
// Then I picked tango because it has a lot of lines, a bright background
// and I like the looks of it.
const defaultLightTheme = "tango"

// Checks the terminal background color and returns either a dark or light theme
func GetStyleForScreen(screen twin.Screen) chroma.Style {
	var style = *styles.Get(defaultDarkTheme)

	t0 := time.Now()
	screen.RequestTerminalBackgroundColor()
	select {
	case event := <-screen.Events():
		// Event received, let's see if it's the one we want
		switch ev := event.(type) {

		case twin.EventTerminalBackgroundDetected:
			log.Debug("Terminal background color detected as ", ev.Color, " after ", time.Since(t0))

			distanceToBlack := ev.Color.Distance(twin.NewColor24Bit(0, 0, 0))
			distanceToWhite := ev.Color.Distance(twin.NewColor24Bit(255, 255, 255))
			if distanceToBlack < distanceToWhite {
				style = *styles.Get(defaultDarkTheme)
			} else {
				style = *styles.Get(defaultLightTheme)
			}

		default:
			log.Debugf("Expected terminal background color event but got %#v after %s, putting back and giving up", ev, time.Since(t0))
			screen.Events() <- event
		}

	// The worst number I have measured was around 15ms, in GNOME Terminal
	// running inside of VirtualBox. 3x that should be enough for everyone
	// (TM).
	case <-time.After(50 * time.Millisecond):
		log.Debug("Terminal background color still not detected after ", time.Since(t0), ", giving up")
	}

	return style
}
