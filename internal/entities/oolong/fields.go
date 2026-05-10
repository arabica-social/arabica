package oolong

import "fmt"

func teaField(e any, field string) (string, bool) {
	t, ok := e.(*Tea)
	if !ok || t == nil {
		return "", false
	}
	switch field {
	case "name":
		return t.Name, true
	case "category":
		return t.Category, true
	case "sub_style":
		return t.SubStyle, true
	case "origin":
		return t.Origin, true
	case "cultivar":
		return t.Cultivar, true
	case "farm":
		return t.Farm, true
	case "description":
		return t.Description, true
	}
	return "", false
}

func vendorField(e any, field string) (string, bool) {
	v, ok := e.(*Vendor)
	if !ok || v == nil {
		return "", false
	}
	switch field {
	case "name":
		return v.Name, true
	case "location":
		return v.Location, true
	case "website":
		return v.Website, true
	case "description":
		return v.Description, true
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
	case "style":
		return b.Style, true
	case "material":
		return b.Material, true
	case "description":
		return b.Description, true
	case "capacity_ml":
		if b.CapacityMl > 0 {
			return fmt.Sprintf("%d", b.CapacityMl), true
		}
		return "", false
	}
	return "", false
}

func cafeField(e any, field string) (string, bool) {
	c, ok := e.(*Cafe)
	if !ok || c == nil {
		return "", false
	}
	switch field {
	case "name":
		return c.Name, true
	case "location":
		return c.Location, true
	case "address":
		return c.Address, true
	case "website":
		return c.Website, true
	case "description":
		return c.Description, true
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
	case "style":
		return r.Style, true
	case "notes":
		return r.Notes, true
	case "leaf_grams":
		if r.LeafGrams > 0 {
			return fmt.Sprintf("%.1f", r.LeafGrams), true
		}
		return "", false
	}
	return "", false
}

func drinkField(e any, field string) (string, bool) {
	d, ok := e.(*Drink)
	if !ok || d == nil {
		return "", false
	}
	switch field {
	case "name":
		return d.Name, true
	case "style":
		return d.Style, true
	case "description":
		return d.Description, true
	case "tasting_notes":
		return d.TastingNotes, true
	}
	return "", false
}
