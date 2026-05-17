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
	case "origin":
		return t.Origin, true
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

func vesselField(e any, field string) (string, bool) {
	v, ok := e.(*Vessel)
	if !ok || v == nil {
		return "", false
	}
	switch field {
	case "name":
		return v.Name, true
	case "style":
		return v.Style, true
	case "material":
		return v.Material, true
	case "description":
		return v.Description, true
	case "capacity_ml":
		if v.CapacityMl > 0 {
			return fmt.Sprintf("%d", v.CapacityMl), true
		}
		return "", false
	}
	return "", false
}

func infuserField(e any, field string) (string, bool) {
	i, ok := e.(*Infuser)
	if !ok || i == nil {
		return "", false
	}
	switch field {
	case "name":
		return i.Name, true
	case "style":
		return i.Style, true
	case "material":
		return i.Material, true
	case "description":
		return i.Description, true
	}
	return "", false
}
