package explore

import (
	"fmt"
	"strings"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// Sort names supported by the Explore query API.
const (
	SortRecent     = "recent"
	SortPopular    = "popular"
	SortRatingHigh = "rating_high"
)

// FilterKind describes how a filter is represented in explore_values.
type FilterKind string

const (
	FilterFacetText FilterKind = "facet_text"
	FilterNumberMin FilterKind = "number_min"
	FilterNumberMax FilterKind = "number_max"
	FilterBool      FilterKind = "bool"
)

// FilterDef constrains a user-visible filter to a known derived value field.
type FilterDef struct {
	Name  string
	Label string
	Kind  FilterKind
	Field string
}

// Value is a derived searchable/facetable value for one Explore document.
type Value struct {
	Field string
	Text  string
	Num   *float64
}

// Document is the derived representation stored in explore_documents.
type Document struct {
	Title      string
	Summary    string
	SearchText string
	OwnRating  *float64
	ClusterKey string
	SourceRef  string
	Values     []Value
}

// TypeDef defines how one record type participates in Explore.
type TypeDef struct {
	App        string
	RecordType lexicons.RecordType
	NSID       string
	Label      string
	Filters    []FilterDef
	Extract    func(record map[string]any, refs RefLookup) Document
}

// RefLookup returns a referenced raw record by AT-URI. It is intentionally
// shallow: Explore V1 is structured discovery over direct fields and selected
// first-hop references, not arbitrary graph search.
type RefLookup func(uri string) (collection string, record map[string]any, ok bool)

// Registry is the constrained Explore metadata registry.
type Registry struct {
	byNSID map[string]TypeDef
	byType map[lexicons.RecordType]TypeDef
}

func NewArabicaRegistry() *Registry {
	types := []TypeDef{
		{
			App: "arabica", RecordType: lexicons.RecordTypeBean, NSID: arabica.NSIDBean, Label: "Beans",
			Filters: []FilterDef{{"origin", "Origin", FilterFacetText, "origin"}, {"variety", "Variety", FilterFacetText, "variety"}, {"process", "Process", FilterFacetText, "process"}, {"roast_level", "Roast level", FilterFacetText, "roast_level"}, {"roaster", "Roaster", FilterFacetText, "roaster"}, {"min_rating", "Minimum rating", FilterNumberMin, "rating"}, {"closed", "Closed", FilterBool, "closed"}},
			Extract: extractBean,
		},
		{
			App: "arabica", RecordType: lexicons.RecordTypeRoaster, NSID: arabica.NSIDRoaster, Label: "Roasters",
			Filters: []FilterDef{{"location", "Location", FilterFacetText, "location"}},
			Extract: extractRoaster,
		},
		{
			App: "arabica", RecordType: lexicons.RecordTypeGrinder, NSID: arabica.NSIDGrinder, Label: "Grinders",
			Filters: []FilterDef{{"grinder_type", "Grinder type", FilterFacetText, "grinder_type"}, {"burr_type", "Burr type", FilterFacetText, "burr_type"}},
			Extract: extractGrinder,
		},
		{
			App: "arabica", RecordType: lexicons.RecordTypeBrewer, NSID: arabica.NSIDBrewer, Label: "Brewers",
			Filters: []FilterDef{{"brewer_type", "Brewer type", FilterFacetText, "brewer_type"}},
			Extract: extractBrewer,
		},
		{
			App: "arabica", RecordType: lexicons.RecordTypeRecipe, NSID: arabica.NSIDRecipe, Label: "Recipes",
			Filters: []FilterDef{{"brewer_type", "Brewer type", FilterFacetText, "brewer_type"}, {"ratio_min", "Ratio minimum", FilterNumberMin, "ratio"}, {"ratio_max", "Ratio maximum", FilterNumberMax, "ratio"}},
			Extract: extractRecipe,
		},
	}
	r := &Registry{byNSID: make(map[string]TypeDef), byType: make(map[lexicons.RecordType]TypeDef)}
	for _, t := range types {
		r.byNSID[t.NSID] = t
		r.byType[t.RecordType] = t
	}
	return r
}

func (r *Registry) TypeByNSID(nsid string) (TypeDef, bool)      { t, ok := r.byNSID[nsid]; return t, ok }
func (r *Registry) Type(rt lexicons.RecordType) (TypeDef, bool) { t, ok := r.byType[rt]; return t, ok }
func (r *Registry) Types() []TypeDef {
	out := make([]TypeDef, 0, len(r.byType))
	for _, t := range r.byType {
		out = append(out, t)
	}
	return out
}
func (r *Registry) ValidateSort(sort string) string {
	switch sort {
	case SortPopular, SortRatingHigh:
		return sort
	default:
		return SortRecent
	}
}
func (r *Registry) ValidateFilter(recordType lexicons.RecordType, name string) (FilterDef, bool) {
	if t, ok := r.Type(recordType); ok {
		for _, f := range t.Filters {
			if f.Name == name {
				return f, true
			}
		}
	}
	for _, t := range r.byType {
		for _, f := range t.Filters {
			if f.Name == name {
				return f, true
			}
		}
	}
	return FilterDef{}, false
}

func extractBean(rec map[string]any, refs RefLookup) Document {
	name, origin := str(rec, "name"), str(rec, "origin")
	title := firstNonEmpty(name, origin, "Coffee bean")
	vals := valuesText("origin", origin, "variety", str(rec, "variety"), "process", str(rec, "process"), "roast_level", str(rec, "roastLevel"))
	if roasterRef := str(rec, "roasterRef"); roasterRef != "" {
		_, rr, ok := refs(roasterRef)
		if ok {
			vals = appendText(vals, "roaster", str(rr, "name"))
		}
	}
	var rating *float64
	if v, ok := num(rec, "rating"); ok {
		rating = &v
		vals = appendNum(vals, "rating", v)
	}
	if b, ok := rec["closed"].(bool); ok {
		if b {
			vals = appendText(vals, "closed", "true")
		} else {
			vals = appendText(vals, "closed", "false")
		}
	}
	search := joinSearch(title, origin, str(rec, "description"), str(rec, "variety"), str(rec, "process"), str(rec, "roastLevel"), textValue(vals, "roaster"))
	return doc(rec, title, str(rec, "description"), search, rating, vals)
}
func extractRoaster(rec map[string]any, refs RefLookup) Document {
	vals := valuesText("location", str(rec, "location"))
	return doc(rec, firstNonEmpty(str(rec, "name"), "Roaster"), str(rec, "location"), joinSearch(str(rec, "name"), str(rec, "location"), str(rec, "website")), nil, vals)
}
func extractGrinder(rec map[string]any, refs RefLookup) Document {
	gt, bt := normalizeGrinderType(str(rec, "grinderType")), normalizeBurrType(str(rec, "burrType"))
	vals := valuesText("grinder_type", gt, "burr_type", bt)
	return doc(rec, firstNonEmpty(str(rec, "name"), "Grinder"), str(rec, "notes"), joinSearch(str(rec, "name"), gt, bt, str(rec, "notes")), nil, vals)
}
func extractBrewer(rec map[string]any, refs RefLookup) Document {
	bt := str(rec, "brewerType")
	vals := valuesText("brewer_type", bt)
	return doc(rec, firstNonEmpty(str(rec, "name"), "Brewer"), str(rec, "description"), joinSearch(str(rec, "name"), bt, str(rec, "description")), nil, vals)
}
func extractRecipe(rec map[string]any, refs RefLookup) Document {
	bt := str(rec, "brewerType")
	if bt == "" {
		if _, br, ok := refs(str(rec, "brewerRef")); ok {
			bt = str(br, "brewerType")
		}
	}
	vals := valuesText("brewer_type", bt)
	coffee, cok := num(rec, "coffeeAmount")
	water, wok := num(rec, "waterAmount")
	if cok {
		coffee /= 10
	}
	if wok {
		water /= 10
	}
	if cok && wok && coffee > 0 {
		vals = appendNum(vals, "ratio", water/coffee)
	}
	return doc(rec, firstNonEmpty(str(rec, "name"), "Recipe"), str(rec, "notes"), joinSearch(str(rec, "name"), bt, str(rec, "notes")), nil, vals)
}

func doc(rec map[string]any, title, summary, search string, rating *float64, vals []Value) Document {
	src := str(rec, "sourceRef")
	return Document{Title: title, Summary: summary, SearchText: strings.ToLower(search), OwnRating: rating, SourceRef: src, Values: vals}
}
func str(m map[string]any, key string) string { v, _ := m[key].(string); return strings.TrimSpace(v) }
func num(m map[string]any, key string) (float64, bool) {
	switch v := m[key].(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
func joinSearch(vals ...string) string {
	var out []string
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return strings.Join(out, " ")
}
func valuesText(kv ...string) []Value {
	var out []Value
	for i := 0; i+1 < len(kv); i += 2 {
		out = appendText(out, kv[i], kv[i+1])
	}
	return out
}
func appendText(vals []Value, field, text string) []Value {
	if text = strings.TrimSpace(text); text != "" {
		vals = append(vals, Value{Field: field, Text: text})
	}
	return vals
}
func appendNum(vals []Value, field string, n float64) []Value {
	return append(vals, Value{Field: field, Num: &n})
}
func textValue(vals []Value, field string) string {
	for _, v := range vals {
		if v.Field == field {
			return v.Text
		}
	}
	return ""
}
func normalizeGrinderType(v string) string {
	switch v {
	case "hand":
		return "Hand"
	case "electric":
		return "Electric"
	case "portable_electric":
		return "Portable Electric"
	default:
		return v
	}
}
func normalizeBurrType(v string) string {
	switch v {
	case "conical":
		return "Conical"
	case "flat":
		return "Flat"
	case "blade":
		return "Blade"
	default:
		return v
	}
}

func (f FilterDef) String() string { return fmt.Sprintf("%s:%s", f.Name, f.Kind) }
