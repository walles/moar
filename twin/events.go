package twin

type Event interface {
	// This interface will be blank until further notice
}

type EventRune struct {
	rune rune
}

type EventKeyCode struct {
	keyCode KeyCode
}

type MouseButtonMask uint16

const (
	MouseWheelUp MouseButtonMask = 1 << iota
	MouseWheelDown
	MouseWheelLeft
	MouseWheelRight
)

type EventMouse struct {
	buttons MouseButtonMask
}

// After you get this, query Screen.Size() to get the new size
type EventResize struct {
	// This interface intentionally left blank
}

func (eventRune *EventRune) Rune() rune {
	return eventRune.rune
}

func (eventKeyCode *EventKeyCode) KeyCode() KeyCode {
	return eventKeyCode.keyCode
}

func (eventMouse *EventMouse) Buttons() MouseButtonMask {
	return eventMouse.buttons
}
