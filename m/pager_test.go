package m

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell"
)

func TestUnicodeRendering(t *testing.T) {
	reader, err := NewReaderFromStream(strings.NewReader("åäö"))
	if err != nil {
		panic(err)
	}

	screen := tcell.NewSimulationScreen("UTF-8")
	pager := NewPager(*reader)
	pager.Quit()
	pager.StartPaging(screen)

	contents, _, _ := screen.GetContents()
	s := ""
	for i := 0; i <= 2; i++ {
		cell := contents[i]
		for _, r := range cell.Runes {
			s += string(r)
		}
	}

	if s != "åäö" {
		t.Errorf("Expected 'åäö', got: <%s>", s)
	}
}
