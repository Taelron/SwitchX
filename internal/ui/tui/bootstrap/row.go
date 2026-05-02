package bootstrap

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/Taelron/SwitchX/internal/ui/tui/styles"
)

// trimmed returns the textinput's value with leading/trailing whitespace
// stripped. Used by every step so accidental spaces don't leak into the
// persisted config or the validate roundtrip.
func trimmed(t textinput.Model) string {
	return strings.TrimSpace(t.Value())
}

// Column widths shared by every wizard step. They sum to the modal's
// content width so each row fills exactly one line and the right-hand
// error column stays at a fixed offset across steps and focus changes.
//
// labelColumnWidth must accommodate the longest label ("Subscription:"
// = 13 chars on step 2) plus a small breathing margin so lipgloss does
// not wrap it onto two lines.
const (
	markerColumnWidth = 2
	labelColumnWidth  = 16
	widgetColumnWidth = 26
	errorColumnWidth  = styles.ModalContentWidth - markerColumnWidth - labelColumnWidth - widgetColumnWidth
	textInputWidth    = widgetColumnWidth - 2
)

// fieldRow is the data needed to render one row of a wizard step.
type fieldRow struct {
	focused bool
	label   string
	widget  string
	err     string
}

// renderRow draws one fieldRow at the shared column layout. Static rows
// (e.g. the read-only provider label on step 2) pass focused=false and
// err="" so the focus marker is absent and the error column renders
// blank.
func renderRow(r fieldRow) string {
	marker := "  "
	if r.focused {
		marker = styles.FocusedMarker.Render("▸ ")
	}
	return lipgloss.JoinHorizontal(lipgloss.Top,
		marker,
		styles.FieldLabel.Width(labelColumnWidth).Render(r.label+":"),
		lipgloss.NewStyle().Width(widgetColumnWidth).Render(r.widget),
		styles.FieldError.Width(errorColumnWidth).PaddingLeft(2).Render(r.err),
	)
}
