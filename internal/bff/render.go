package bff

import (
	"html/template"
	"net/http"
	"os"
	"sync"

	"arabica/internal/feed"
	"arabica/internal/models"
)

var (
	templates     *template.Template
	templatesOnce sync.Once
	templatesErr  error
)

// loadTemplates initializes templates lazily - only when first needed
func loadTemplates() error {
	templatesOnce.Do(func() {
		// Parse all template files including partials
		templates = template.New("")
		templates = templates.Funcs(template.FuncMap{
			"formatTemp":       FormatTemp,
			"formatTime":       FormatTime,
			"formatRating":     FormatRating,
			"formatID":         FormatID,
			"formatInt":        FormatInt,
			"formatRoasterID":  FormatRoasterID,
			"poursToJSON":      PoursToJSON,
			"ptrEquals":        PtrEquals[int],
			"ptrValue":         PtrValue[int],
			"iterate":          Iterate,
			"iterateRemaining": IterateRemaining,
			"hasTemp":          HasTemp,
			"hasValue":         HasValue,
		})

		// Try to find templates relative to working directory
		// This supports both running from project root and from package directory
		paths := []string{
			"templates/*.tmpl",
			"../../templates/*.tmpl",    // for when tests run from internal/bff
			"../../../templates/*.tmpl", // for deeper test directories
		}

		var err error
		for _, path := range paths {
			dir := path[:len(path)-6] // Remove *.tmpl
			if _, statErr := os.Stat(dir); statErr == nil {
				templates, err = templates.ParseGlob(path)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			templatesErr = err
			return
		}

		// Parse partials
		partialPaths := []string{
			"templates/partials/*.tmpl",
			"../../templates/partials/*.tmpl",
			"../../../templates/partials/*.tmpl",
		}

		for _, path := range partialPaths {
			dir := path[:len(path)-6]
			if _, statErr := os.Stat(dir); statErr == nil {
				templates, err = templates.ParseGlob(path)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			templatesErr = err
		}
	})
	return templatesErr
}

// PageData contains data for rendering pages
type PageData struct {
	Title           string
	Beans           []*models.Bean
	Roasters        []*models.Roaster
	Grinders        []*models.Grinder
	Brewers         []*models.Brewer
	Brew            *BrewData
	Brews           []*BrewListData
	FeedItems       []*feed.FeedItem
	IsAuthenticated bool
	UserDID         string
}

// BrewData wraps a brew with pre-serialized JSON for pours
type BrewData struct {
	*models.Brew
	PoursJSON string
}

// BrewListData wraps a brew with pre-formatted display values
type BrewListData struct {
	*models.Brew
	TempFormatted   string
	TimeFormatted   string
	RatingFormatted string
}

// RenderTemplate renders a template with layout
func RenderTemplate(w http.ResponseWriter, tmpl string, data *PageData) error {
	if err := loadTemplates(); err != nil {
		return err
	}
	// Execute the layout template which calls the content template
	return templates.ExecuteTemplate(w, "layout", data)
}

// RenderHome renders the home page
func RenderHome(w http.ResponseWriter, isAuthenticated bool, userDID string, feedItems []*feed.FeedItem) error {
	if err := loadTemplates(); err != nil {
		return err
	}
	data := &PageData{
		Title:           "Home",
		IsAuthenticated: isAuthenticated,
		UserDID:         userDID,
		FeedItems:       feedItems,
	}
	// Need to execute layout with the home template
	t := template.Must(templates.Clone())
	t = template.Must(t.ParseFiles(findTemplatePath("home.tmpl")))
	return t.ExecuteTemplate(w, "layout", data)
}

// RenderBrewList renders the brew list page
func RenderBrewList(w http.ResponseWriter, brews []*models.Brew, isAuthenticated bool, userDID string) error {
	if err := loadTemplates(); err != nil {
		return err
	}
	brewList := make([]*BrewListData, len(brews))
	for i, brew := range brews {
		brewList[i] = &BrewListData{
			Brew:            brew,
			TempFormatted:   FormatTemp(brew.Temperature),
			TimeFormatted:   FormatTime(brew.TimeSeconds),
			RatingFormatted: FormatRating(brew.Rating),
		}
	}

	data := &PageData{
		Title:           "All Brews",
		Brews:           brewList,
		IsAuthenticated: isAuthenticated,
		UserDID:         userDID,
	}
	t := template.Must(templates.Clone())
	t = template.Must(t.ParseFiles(findTemplatePath("brew_list.tmpl")))
	return t.ExecuteTemplate(w, "layout", data)
}

// RenderBrewForm renders the brew form page
func RenderBrewForm(w http.ResponseWriter, beans []*models.Bean, roasters []*models.Roaster, grinders []*models.Grinder, brewers []*models.Brewer, brew *models.Brew, isAuthenticated bool, userDID string) error {
	if err := loadTemplates(); err != nil {
		return err
	}
	var brewData *BrewData
	title := "New Brew"

	if brew != nil {
		title = "Edit Brew"
		brewData = &BrewData{
			Brew:      brew,
			PoursJSON: PoursToJSON(brew.Pours),
		}
	}

	data := &PageData{
		Title:           title,
		Beans:           beans,
		Roasters:        roasters,
		Grinders:        grinders,
		Brewers:         brewers,
		Brew:            brewData,
		IsAuthenticated: isAuthenticated,
		UserDID:         userDID,
	}
	t := template.Must(templates.Clone())
	t = template.Must(t.ParseFiles(findTemplatePath("brew_form.tmpl")))
	return t.ExecuteTemplate(w, "layout", data)
}

// RenderManage renders the manage page
func RenderManage(w http.ResponseWriter, beans []*models.Bean, roasters []*models.Roaster, grinders []*models.Grinder, brewers []*models.Brewer, isAuthenticated bool, userDID string) error {
	if err := loadTemplates(); err != nil {
		return err
	}
	data := &PageData{
		Title:           "Manage",
		Beans:           beans,
		Roasters:        roasters,
		Grinders:        grinders,
		Brewers:         brewers,
		IsAuthenticated: isAuthenticated,
		UserDID:         userDID,
	}
	t := template.Must(templates.Clone())
	t = template.Must(t.ParseFiles(findTemplatePath("manage.tmpl")))
	return t.ExecuteTemplate(w, "layout", data)
}

// findTemplatePath finds the correct path to a template file
func findTemplatePath(name string) string {
	paths := []string{
		"templates/" + name,
		"../../templates/" + name,
		"../../../templates/" + name,
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	// Return the default path even if it doesn't exist - will fail at parse time
	return "templates/" + name
}
