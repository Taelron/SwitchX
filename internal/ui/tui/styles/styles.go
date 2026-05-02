// Package styles centralises Lipgloss colour and style definitions used
// across switchx's TUI. Colours follow @UI Patterns (Taelron Baselines).
//
// New views compose the exported styles instead of redefining colours
// inline so that a future palette change is a single-file edit.
package styles

import "github.com/charmbracelet/lipgloss"

// Colour palette per @UI Patterns. ANSI 256-colour codes so the look
// degrades gracefully on non-truecolor terminals.
var (
	ColorError    = lipgloss.Color("9")   // red
	ColorInfo     = lipgloss.Color("14")  // cyan
	ColorBorder   = lipgloss.Color("12")  // blue
	ColorHint     = lipgloss.Color("241") // dim grey
	ColorEmphasis = lipgloss.Color("15")  // bright white
)

// ModalWidth is the fixed inner width (including padding) of every
// modal box. Pinning it stops the box jumping size as fields focus or
// hint-bar content changes between steps.
const ModalWidth = 80

// Modal is the centred bordered container for wizard steps and modal
// overlays. Padding gives breathing room between the border and content;
// the fixed Width keeps the chrome stable across focus/step changes.
var Modal = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(ColorBorder).
	Padding(1, 2).
	Width(ModalWidth)

// Breadcrumb labels the wizard location at the top of the modal.
var Breadcrumb = lipgloss.NewStyle().
	Foreground(ColorInfo).
	Bold(true).
	MarginBottom(1)

// HintBar renders the keymap reminders along the bottom of a view.
var HintBar = lipgloss.NewStyle().
	Foreground(ColorHint).
	MarginTop(1)

// FieldLabel prefixes each input field.
var FieldLabel = lipgloss.NewStyle().
	Foreground(ColorEmphasis).
	Bold(true)

// FieldError renders inline validation messages beneath a field.
var FieldError = lipgloss.NewStyle().
	Foreground(ColorError)

// FocusedMarker is the leading glyph rendered next to the focused field.
var FocusedMarker = lipgloss.NewStyle().
	Foreground(ColorInfo).
	Bold(true)
