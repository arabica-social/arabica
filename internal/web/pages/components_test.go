package pages

import (
	"arabica/internal/web/components"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
)

// TestButtonComponents tests button component rendering
func TestButtonComponents(t *testing.T) {
	ctx := context.Background()

	t.Run("PrimaryButton", func(t *testing.T) {
		props := components.ButtonProps{
			Text: "Click Me",
			Type: "submit",
		}

		html := renderToString(t, ctx, components.PrimaryButton(props))

		assert.Contains(t, html, "btn-primary")
		assert.Contains(t, html, "Click Me")
		assert.Contains(t, html, `type="submit"`)
	})

	t.Run("SecondaryButton", func(t *testing.T) {
		props := components.ButtonProps{
			Text: "Cancel",
		}

		html := renderToString(t, ctx, components.SecondaryButton(props))

		assert.Contains(t, html, "btn-secondary")
		assert.Contains(t, html, "Cancel")
	})

	t.Run("BackButton", func(t *testing.T) {
		html := renderToString(t, ctx, components.BackButton())

		assert.Contains(t, html, `type="button"`)
		assert.Contains(t, html, `@click="history.back()"`)
		assert.Contains(t, html, "<svg")
		assert.Contains(t, html, "Go back")
	})
}

// TestFormComponents tests form input component rendering
func TestFormComponents(t *testing.T) {
	ctx := context.Background()

	t.Run("TextInput", func(t *testing.T) {
		props := components.TextInputProps{
			Name:        "username",
			Value:       "john",
			Placeholder: "Enter username",
			Required:    true,
		}

		html := renderToString(t, ctx, components.TextInput(props))

		assert.Contains(t, html, `name="username"`)
		assert.Contains(t, html, `value="john"`)
		assert.Contains(t, html, `placeholder="Enter username"`)
		assert.Contains(t, html, "required")
	})

	t.Run("NumberInput", func(t *testing.T) {
		props := components.NumberInputProps{
			Name:        "amount",
			Value:       "18",
			Step:        "0.1",
			Min:         "0",
			Placeholder: "Enter amount",
		}

		html := renderToString(t, ctx, components.NumberInput(props))

		assert.Contains(t, html, `type="number"`)
		assert.Contains(t, html, `step="0.1"`)
		assert.Contains(t, html, `min="0"`)
	})

	t.Run("TextArea", func(t *testing.T) {
		props := components.TextAreaProps{
			Name:        "notes",
			Value:       "Test notes",
			Placeholder: "Enter notes",
			Rows:        5,
		}

		html := renderToString(t, ctx, components.TextArea(props))

		assert.Contains(t, html, `name="notes"`)
		assert.Contains(t, html, "Test notes")
		assert.Contains(t, html, `rows="5"`)
	})

	t.Run("Select", func(t *testing.T) {
		props := components.SelectProps{
			Name:        "choice",
			Placeholder: "Select option",
			Options: []components.SelectOption{
				{Value: "1", Label: "Option 1", Selected: false},
				{Value: "2", Label: "Option 2", Selected: true},
			},
		}

		html := renderToString(t, ctx, components.Select(props))

		assert.Contains(t, html, `name="choice"`)
		assert.Contains(t, html, "Option 1")
		assert.Contains(t, html, "Option 2")
		assert.Contains(t, html, "selected")
	})
}

// TestCardComponent tests card component rendering
func TestCardComponent(t *testing.T) {
	ctx := context.Background()

	t.Run("Basic card", func(t *testing.T) {
		props := components.CardProps{
			InnerCard: false,
		}

		content := templ.Raw("<p>Card content</p>")

		html := renderToString(t, ctx, components.Card(props, content))

		assert.Contains(t, html, "card")
		assert.Contains(t, html, "Card content")
	})

	t.Run("Inner card", func(t *testing.T) {
		props := components.CardProps{
			InnerCard: true,
		}

		content := templ.Raw("<p>Card content</p>")

		html := renderToString(t, ctx, components.Card(props, content))

		assert.Contains(t, html, "card-inner")
	})
}

// Helper function to render a component to string for testing
func renderToString(t *testing.T, ctx context.Context, component templ.Component) string {
	t.Helper()

	var buf strings.Builder
	if err := component.Render(ctx, &buf); err != nil {
		t.Fatalf("Failed to render component: %v", err)
	}

	return buf.String()
}
