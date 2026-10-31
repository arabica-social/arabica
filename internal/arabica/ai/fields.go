package ai

import (
	"encoding/json"
	"errors"
	"math"
	"net/url"
	"strings"
	"time"
)

// Field rules apply to incomplete drafts, not create-record required fields.
// Names and units match the existing JSON mutation API, never lexicon scaling.
type fieldRule struct {
	Type    string  `json:"type"`
	Max     float64 `json:"max,omitempty"`
	Min     float64 `json:"min,omitempty"`
	Integer bool    `json:"integer,omitempty"`
}

func text(max float64) fieldRule { return fieldRule{Type: "string", Max: max} }
func num(max float64, integer bool) fieldRule {
	return fieldRule{Type: "number", Max: max, Integer: integer}
}

// Browser/API numbers stay exactly representable; this is not a brewing-time limit.
const maxSafeNumber = 9007199254740991

var fields = map[string]map[string]fieldRule{
	"brew": {
		"bean_rkey": {Type: "reference"}, "grinder_rkey": {Type: "reference"}, "brewer_rkey": {Type: "reference"},
		"recipe_rkey": {Type: "reference"}, "recipe_owner_did": {Type: "empty"},
		"method": text(100), "temperature": num(212, false), "water_amount": num(10000, true), "coffee_amount": num(1000, true), "time_seconds": num(maxSafeNumber, true), "rating": num(10, true), "grind_size": text(100), "tasting_notes": text(2000),
		"espresso_params.yield_weight": num(maxSafeNumber, false), "espresso_params.pressure": num(maxSafeNumber, false), "espresso_params.pre_infusion_seconds": num(maxSafeNumber, true),
		"pourover_params.bloom_water": num(maxSafeNumber, true), "pourover_params.bloom_seconds": num(maxSafeNumber, true), "pourover_params.drawdown_seconds": num(maxSafeNumber, true), "pourover_params.bypass_water": num(maxSafeNumber, true), "pourover_params.filter": text(100), "pours": {Type: "pours"},
	},
	"bean":    {"name": text(200), "origin": text(200), "variety": text(200), "roast_level": text(100), "roast_date": text(10), "process": text(100), "description": text(2000), "notes": text(2000), "link": text(500), "roaster_rkey": {Type: "reference"}, "rating": {Type: "number", Min: 1, Max: 10, Integer: true}, "closed": {Type: "boolean"}, "source_ref": {Type: "empty"}},
	"grinder": {"name": text(200), "grinder_type": text(50), "burr_type": text(50), "notes": text(2000), "link": text(500), "source_ref": {Type: "empty"}},
	"brewer":  {"name": text(200), "brewer_type": text(100), "description": text(2000), "link": text(500), "source_ref": {Type: "empty"}},
}
var errFields = errors.New("invalid draft fields")

func validateFields(kind string, patch map[string]any) error {
	rules, ok := fields[kind]
	if !ok || len(patch) > len(rules) {
		return errFields
	}
	for key, value := range patch {
		rule, ok := rules[key]
		if !ok {
			return errFields
		}
		if value == nil {
			continue
		}
		switch rule.Type {
		case "string", "reference", "empty":
			s, ok := value.(string)
			if !ok {
				return errFields
			}
			if rule.Type == "empty" && s != "" {
				return errFields
			}
			if rule.Type == "reference" {
				if len(s) > 256 {
					return errFields
				}
				continue
			}
			if len(s) > int(rule.Max) {
				return errFields
			}
			if key == "roast_date" && s != "" {
				if _, err := time.Parse(time.DateOnly, s); err != nil {
					return errFields
				}
			}
			if key == "link" && s != "" {
				u, err := url.Parse(s)
				if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
					return errFields
				}
			}
		case "number":
			n, ok := value.(float64)
			if !ok || math.IsNaN(n) || math.IsInf(n, 0) || n < rule.Min || n > rule.Max || (rule.Integer && math.Trunc(n) != n) {
				return errFields
			}
		case "boolean":
			if _, ok := value.(bool); !ok {
				return errFields
			}
		case "pours":
			arr, ok := value.([]any)
			if !ok || len(arr) > 100 {
				return errFields
			}
			for _, item := range arr {
				p, ok := item.(map[string]any)
				if !ok || len(p) != 2 {
					return errFields
				}
				water, wok := p["water_amount"].(float64)
				seconds, sok := p["time_seconds"].(float64)
				if !wok || !sok || water < 0 || water > maxSafeNumber || seconds < 0 || seconds > maxSafeNumber || math.Trunc(water) != water || math.Trunc(seconds) != seconds {
					return errFields
				}
			}
		default:
			return errFields
		}
	}
	return nil
}

func meaningful(patch map[string]any) bool {
	for _, v := range patch {
		switch value := v.(type) {
		case string:
			if strings.TrimSpace(value) != "" {
				return true
			}
		case []any:
			if len(value) > 0 {
				return true
			}
		case float64:
			if value > 0 {
				return true
			}
		case bool:
			if value {
				return true
			}
		}
	}
	return false
}

// decodeStrict consumes the entire JSON value. Unknown operation/envelope fields
// cannot smuggle model-authored actions through a partially decoded response.
func decodeStrict(data []byte, target any) error {
	d := json.NewDecoder(strings.NewReader(string(data)))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if d.Decode(new(any)) == nil {
		return errors.New("trailing data")
	}
	if !json.Valid(data) {
		return errors.New("invalid JSON")
	}
	return nil
}
