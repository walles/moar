package m

type reusableStringBuilder struct {
}

func (rsb reusableStringBuilder) Reset() {
	panic("Unimplemented")
}

func (rsb reusableStringBuilder) WriteRune(char rune) {
	panic("Unimplemented")
}

func (rsb reusableStringBuilder) String() string {
	panic("Unimplemented")
}
