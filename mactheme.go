package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// MacTheme is a custom theme for a Mac-like appearance
type MacTheme struct{}

var _ fyne.Theme = (*MacTheme)(nil)

// Color returns the color for the specified theme ColorName
func (m *MacTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 28, G: 28, B: 30, A: 255}
	case theme.ColorNameButton:
		return color.NRGBA{R: 59, G: 59, B: 61, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 60, G: 60, B: 60, A: 128}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 242, G: 242, B: 247, A: 255}
	case theme.ColorNameHover:
		return color.NRGBA{R: 85, G: 85, B: 85, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 180, G: 180, B: 180, A: 255}
	case theme.ColorNamePressed:
		return color.NRGBA{R: 0, G: 122, B: 255, A: 255}
	case theme.ColorNameScrollBar:
		return color.NRGBA{R: 100, G: 100, B: 100, A: 128}
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0, G: 0, B: 0, A: 60}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

// Font returns the font for the specified TextStyle and size
func (m *MacTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

// Icon returns the icon resource for the specified IconName
func (m *MacTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Size returns the size for the specified theme SizeName
func (m *MacTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8
	case theme.SizeNameInlineIcon:
		return 20
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 6
	case theme.SizeNameText:
		return 14
	case theme.SizeNameHeadingText:
		return 18
	case theme.SizeNameSubHeadingText:
		return 16
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameSeparatorThickness:
		return 1
	case theme.SizeNameInnerPadding:
		return 4
	default:
		return theme.DefaultTheme().Size(name)
	}
}
