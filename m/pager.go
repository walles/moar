package m

// Pager is the main on-screen pager
type _Pager struct {
	reader _Reader
}

// NewPager creates a new Pager
func NewPager(r _Reader) *_Pager {
	return &_Pager{
		reader: r,
	}
}

// StartPaging brings up the pager on screen
func (p *_Pager) StartPaging() {

}
