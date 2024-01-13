// Package twin provides Terminal Window interaction
package twin

// Used for testing.
//
// Try GetRow() after some SetCell() calls to see what you got.
type FakeScreen struct {
	width  int
	height int
	cells  [][]Cell
}

func NewFakeScreen(width int, height int) *FakeScreen {
	rows := make([][]Cell, height)
	for i := 0; i < height; i++ {
		rows[i] = make([]Cell, width)
	}

	return &FakeScreen{
		width:  width,
		height: height,
		cells:  rows,
	}
}

func (screen *FakeScreen) Close() {
	// This method intentionally left blank
}

func (screen *FakeScreen) Clear() {
	// This method's contents has been copied from UnixScreen.Clear()

	empty := NewCell(' ', StyleDefault)

	width, height := screen.Size()
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			screen.cells[row][column] = empty
		}
	}
}

func (screen *FakeScreen) SetCell(column int, row int, cell Cell) {
	// This method's contents has been copied from UnixScreen.Clear()

	if column < 0 {
		return
	}
	if row < 0 {
		return
	}

	width, height := screen.Size()
	if column >= width {
		return
	}
	if row >= height {
		return
	}
	screen.cells[row][column] = cell
}

func (screen *FakeScreen) Show() {
	// This method intentionally left blank
}

func (screen *FakeScreen) ShowNLines(int) {
	// This method intentionally left blank
}

func (screen *FakeScreen) Size() (width int, height int) {
	return screen.width, screen.height
}

func (screen *FakeScreen) TerminalBackgroundColor() *Color {
	// THe fake screen has an unknown background color
	return nil
}

func (screen *FakeScreen) ShowCursorAt(_ int, _ int) {
	// This method intentionally left blank
}

func (screen *FakeScreen) Events() chan Event {
	// TODO: Do better here if or when this becomes a problem
	return nil
}

func (screen *FakeScreen) GetRow(row int) []Cell {
	return screen.cells[row]
}
