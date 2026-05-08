package arabica

import (
	"fmt"
)

func beanField(e any, field string) (string, bool) {
	b, ok := e.(*Bean)
	if !ok || b == nil {
		return "", false
	}
	switch field {
	case "name":
		return b.Name, true
	case "origin":
		return b.Origin, true
	case "variety":
		return b.Variety, true
	case "process":
		return b.Process, true
	case "description":
		return b.Description, true
	}
	return "", false
}

func roasterField(e any, field string) (string, bool) {
	r, ok := e.(*Roaster)
	if !ok || r == nil {
		return "", false
	}
	switch field {
	case "name":
		return r.Name, true
	case "location":
		return r.Location, true
	case "website":
		return r.Website, true
	}
	return "", false
}

func grinderField(e any, field string) (string, bool) {
	g, ok := e.(*Grinder)
	if !ok || g == nil {
		return "", false
	}
	switch field {
	case "name":
		return g.Name, true
	case "notes":
		return g.Notes, true
	}
	return "", false
}

func brewerField(e any, field string) (string, bool) {
	b, ok := e.(*Brewer)
	if !ok || b == nil {
		return "", false
	}
	switch field {
	case "name":
		return b.Name, true
	case "brewer_type":
		return b.BrewerType, true
	case "description":
		return b.Description, true
	}
	return "", false
}

func recipeField(e any, field string) (string, bool) {
	r, ok := e.(*Recipe)
	if !ok || r == nil {
		return "", false
	}
	switch field {
	case "name":
		return r.Name, true
	case "brewer_type":
		return r.BrewerType, true
	case "notes":
		return r.Notes, true
	case "coffee_amount":
		if r.CoffeeAmount > 0 {
			return fmt.Sprintf("%.1f", r.CoffeeAmount), true
		}
		return "", false
	case "water_amount":
		if r.WaterAmount > 0 {
			return fmt.Sprintf("%.1f", r.WaterAmount), true
		}
		return "", false
	}
	return "", false
}
