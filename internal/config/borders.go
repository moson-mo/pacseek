package config

import "github.com/rivo/tview"

// Borders contains all border elements of our border style
type Borders struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune

	HorizontalFocus  rune
	VerticalFocus    rune
	TopLeftFocus     rune
	TopRightFocus    rune
	BottomLeftFocus  rune
	BottomRightFocus rune
}

// default scheme
const (
	defaultBorderStyle = "Double"
)

// border style definitions
var (
	borderStyles = map[string]Borders{
		"Double": {
			Horizontal:  tview.BoxDrawingsLightHorizontal,
			Vertical:    tview.BoxDrawingsLightVertical,
			TopLeft:     tview.BoxDrawingsLightDownAndRight,
			TopRight:    tview.BoxDrawingsLightDownAndLeft,
			BottomLeft:  tview.BoxDrawingsLightUpAndRight,
			BottomRight: tview.BoxDrawingsLightUpAndLeft,

			HorizontalFocus:  tview.BoxDrawingsDoubleHorizontal,
			VerticalFocus:    tview.BoxDrawingsDoubleVertical,
			TopLeftFocus:     tview.BoxDrawingsDoubleDownAndRight,
			TopRightFocus:    tview.BoxDrawingsDoubleDownAndLeft,
			BottomLeftFocus:  tview.BoxDrawingsDoubleUpAndRight,
			BottomRightFocus: tview.BoxDrawingsDoubleUpAndLeft,
		},
		"Thick": {
			Horizontal:  tview.BoxDrawingsLightHorizontal,
			Vertical:    tview.BoxDrawingsLightVertical,
			TopLeft:     tview.BoxDrawingsLightDownAndRight,
			TopRight:    tview.BoxDrawingsLightDownAndLeft,
			BottomLeft:  tview.BoxDrawingsLightUpAndRight,
			BottomRight: tview.BoxDrawingsLightUpAndLeft,

			HorizontalFocus:  tview.BoxDrawingsHeavyHorizontal,
			VerticalFocus:    tview.BoxDrawingsHeavyVertical,
			TopLeftFocus:     tview.BoxDrawingsHeavyDownAndRight,
			TopRightFocus:    tview.BoxDrawingsHeavyDownAndLeft,
			BottomLeftFocus:  tview.BoxDrawingsHeavyUpAndRight,
			BottomRightFocus: tview.BoxDrawingsHeavyUpAndLeft,
		},
		"Single": {
			Horizontal:  tview.BoxDrawingsLightHorizontal,
			Vertical:    tview.BoxDrawingsLightVertical,
			TopLeft:     tview.BoxDrawingsLightDownAndRight,
			TopRight:    tview.BoxDrawingsLightDownAndLeft,
			BottomLeft:  tview.BoxDrawingsLightUpAndRight,
			BottomRight: tview.BoxDrawingsLightUpAndLeft,

			HorizontalFocus:  tview.BoxDrawingsLightHorizontal,
			VerticalFocus:    tview.BoxDrawingsLightVertical,
			TopLeftFocus:     tview.BoxDrawingsLightDownAndRight,
			TopRightFocus:    tview.BoxDrawingsLightDownAndLeft,
			BottomLeftFocus:  tview.BoxDrawingsLightUpAndRight,
			BottomRightFocus: tview.BoxDrawingsLightUpAndLeft,
		},
		"Round": {
			Horizontal:  tview.BoxDrawingsLightHorizontal,
			Vertical:    tview.BoxDrawingsLightVertical,
			TopLeft:     tview.BoxDrawingsLightArcDownAndRight,
			TopRight:    tview.BoxDrawingsLightArcDownAndLeft,
			BottomLeft:  tview.BoxDrawingsLightArcUpAndRight,
			BottomRight: tview.BoxDrawingsLightArcUpAndLeft,

			HorizontalFocus:  tview.BoxDrawingsLightHorizontal,
			VerticalFocus:    tview.BoxDrawingsLightVertical,
			TopLeftFocus:     tview.BoxDrawingsLightArcDownAndRight,
			TopRightFocus:    tview.BoxDrawingsLightArcDownAndLeft,
			BottomLeftFocus:  tview.BoxDrawingsLightArcUpAndRight,
			BottomRightFocus: tview.BoxDrawingsLightArcUpAndLeft,
		},
	}
)

func (s *Settings) SetBorderStyle(style string) {
	tview.Borders.Horizontal = borderStyles[style].Horizontal
	tview.Borders.Vertical = borderStyles[style].Vertical
	tview.Borders.TopLeft = borderStyles[style].TopLeft
	tview.Borders.TopRight = borderStyles[style].TopRight
	tview.Borders.BottomLeft = borderStyles[style].BottomLeft
	tview.Borders.BottomRight = borderStyles[style].BottomRight

	tview.Borders.HorizontalFocus = borderStyles[style].HorizontalFocus
	tview.Borders.VerticalFocus = borderStyles[style].VerticalFocus
	tview.Borders.TopLeftFocus = borderStyles[style].TopLeftFocus
	tview.Borders.TopRightFocus = borderStyles[style].TopRightFocus
	tview.Borders.BottomLeftFocus = borderStyles[style].BottomLeftFocus
	tview.Borders.BottomRightFocus = borderStyles[style].BottomRightFocus
}

// Returns all available border styles
func BorderStyles() []string {
	return []string{"Double", "Thick", "Single", "Round"}
}
