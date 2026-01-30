package pages

import (
	"arabica/internal/web/components"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
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

		if !strings.Contains(html, "btn-primary") {
			t.Error("Expected btn-primary class")
		}
		if !strings.Contains(html, "Click Me") {
			t.Error("Expected button text 'Click Me'")
		}
		if !strings.Contains(html, `type="submit"`) {
			t.Error("Expected type=submit")
		}
	})

	t.Run("SecondaryButton", func(t *testing.T) {
		props := components.ButtonProps{
			Text: "Cancel",
		}

		html := renderToString(t, ctx, components.SecondaryButton(props))

		if !strings.Contains(html, "btn-secondary") {
			t.Error("Expected btn-secondary class")
		}
		if !strings.Contains(html, "Cancel") {
			t.Error("Expected button text 'Cancel'")
		}
	})

	t.Run("BackButton", func(t *testing.T) {
		html := renderToString(t, ctx, components.BackButton())

		if !strings.Contains(html, `type="button"`) {
			t.Error("Expected type=button to prevent form submission")
		}
		if !strings.Contains(html, `@click="history.back()"`) {
			t.Error("Expected Alpine.js click handler for back button")
		}
		if !strings.Contains(html, "<svg") {
			t.Error("Expected SVG icon")
		}
		if !strings.Contains(html, "Go back") {
			t.Error("Expected aria-label for accessibility")
		}
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

		if !strings.Contains(html, `name="username"`) {
			t.Error("Expected name attribute")
		}
		if !strings.Contains(html, `value="john"`) {
			t.Error("Expected value attribute")
		}
		if !strings.Contains(html, `placeholder="Enter username"`) {
			t.Error("Expected placeholder attribute")
		}
		if !strings.Contains(html, "required") {
			t.Error("Expected required attribute")
		}
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

		if !strings.Contains(html, `type="number"`) {
			t.Error("Expected type=number")
		}
		if !strings.Contains(html, `step="0.1"`) {
			t.Error("Expected step attribute")
		}
		if !strings.Contains(html, `min="0"`) {
			t.Error("Expected min attribute")
		}
	})

	t.Run("TextArea", func(t *testing.T) {
		props := components.TextAreaProps{
			Name:        "notes",
			Value:       "Test notes",
			Placeholder: "Enter notes",
			Rows:        5,
		}

		html := renderToString(t, ctx, components.TextArea(props))

		if !strings.Contains(html, `name="notes"`) {
			t.Error("Expected name attribute")
		}
		if !strings.Contains(html, "Test notes") {
			t.Error("Expected textarea value")
		}
		if !strings.Contains(html, `rows="5"`) {
			t.Error("Expected rows attribute")
		}
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

		if !strings.Contains(html, `name="choice"`) {
			t.Error("Expected name attribute")
		}
		if !strings.Contains(html, "Option 1") {
			t.Error("Expected Option 1")
		}
		if !strings.Contains(html, "Option 2") {
			t.Error("Expected Option 2")
		}
		// Check that option 2 has selected attribute
		if !strings.Contains(html, "selected") {
			t.Error("Expected selected attribute on option 2")
		}
	})
}

// TestModalComponents tests modal component rendering
func TestModalComponents(t *testing.T) {
	ctx := context.Background()

	t.Run("Modal with static title", func(t *testing.T) {
		props := components.ModalProps{
			Show:  "showModal",
			Title: "Test Modal",
		}

		// Create simple content component
		content := templ.Raw("<p>Modal content</p>")

		html := renderToString(t, ctx, components.Modal(props, content))

		if !strings.Contains(html, `x-show="showModal"`) {
			t.Error("Expected x-show attribute")
		}
		if !strings.Contains(html, "Test Modal") {
			t.Error("Expected modal title")
		}
		if !strings.Contains(html, "modal-backdrop") {
			t.Error("Expected modal-backdrop class")
		}
	})

	t.Run("Modal with dynamic title", func(t *testing.T) {
		props := components.ModalProps{
			Show:      "showModal",
			TitleExpr: "editing ? 'Edit' : 'Add'",
		}

		content := templ.Raw("<p>Modal content</p>")

		html := renderToString(t, ctx, components.Modal(props, content))

		if !strings.Contains(html, `x-text="editing ? &#39;Edit&#39; : &#39;Add&#39;"`) &&
			!strings.Contains(html, `x-text="editing ? 'Edit' : 'Add'"`) {
			t.Error("Expected x-text attribute with dynamic title")
		}
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

		if !strings.Contains(html, "card") {
			t.Error("Expected card class")
		}
		if !strings.Contains(html, "Card content") {
			t.Error("Expected card content")
		}
	})

	t.Run("Inner card", func(t *testing.T) {
		props := components.CardProps{
			InnerCard: true,
		}

		content := templ.Raw("<p>Card content</p>")

		html := renderToString(t, ctx, components.Card(props, content))

		if !strings.Contains(html, "card-inner") {
			t.Error("Expected card-inner class")
		}
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
