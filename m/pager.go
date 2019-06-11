package m

// Pager is the main on-screen pager
type Pager struct {
	reader _Reader
}

// NewPager creates a new Pager
func NewPager(r _Reader) *Pager {
	return &Pager{
		reader: r,
	}
}

// StartPaging brings up the pager on screen
func (p *Pager) StartPaging() {

}
