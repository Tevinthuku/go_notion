package page

type PageConfig struct {
	// defines the spacing in position between pages
	// the position is used to order pages
	Spacing uint
}

func NewPageConfig(spacing uint) *PageConfig {
	return &PageConfig{Spacing: spacing}
}
