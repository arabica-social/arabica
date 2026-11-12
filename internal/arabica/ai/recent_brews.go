package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/pdewey.com/atp"
)

type recentBrew struct {
	ID     string            `json:"id"`
	Date   string            `json:"date"`
	Fields map[string]any    `json:"fields"`
	Labels map[string]string `json:"labels,omitempty"`
	record map[string]any
	key    string
	date   time.Time
}

func loadRecentBrews(ctx context.Context, store LibraryStore) ([]recentBrew, error) {
	rows, err := store.FetchAllRecords(ctx, arabica.NSIDBrew)
	if err != nil {
		return nil, ErrLibrary
	}
	recent := make([]recentBrew, 0, 3)
	for _, row := range rows {
		if !atp.ValidateRKey(row.RKey) || (row.URI != "" && row.URI != "at://"+store.DID()+"/"+arabica.NSIDBrew+"/"+row.RKey) {
			return nil, ErrLibrary
		}
		raw, _ := row.Record["createdAt"].(string)
		date, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			continue
		}
		duplicate := false
		for _, prev := range recent {
			if prev.key == row.RKey {
				duplicate = true
			}
		}
		if duplicate {
			continue
		}
		candidate := recentBrew{Date: date.Format(time.RFC3339Nano), record: row.Record, key: row.RKey, date: date}
		pos := len(recent)
		for i, prev := range recent {
			if date.After(prev.date) || (date.Equal(prev.date) && row.RKey < prev.key) {
				pos = i
				break
			}
		}
		if pos >= 3 {
			continue
		}
		recent = append(recent, candidate)
		copy(recent[pos+1:], recent[pos:])
		recent[pos] = candidate
		if len(recent) > 3 {
			recent = recent[:3]
		}
	}
	for i := range recent {
		recent[i].ID = fmt.Sprintf("recent-brew-%d", i+1)
	}
	return recent, nil
}

func recentPins(recent []recentBrew, did string) map[string]bool {
	pins := map[string]bool{}
	for _, brew := range recent {
		for _, kind := range []string{"bean", "brewer", "grinder", "recipe"} {
			ref, _ := brew.record[kind+"Ref"].(string)
			prefix := "at://" + did + "/" + arabica.NSIDBase + "." + kind + "/"
			if strings.HasPrefix(ref, prefix) && atp.ValidateRKey(strings.TrimPrefix(ref, prefix)) {
				pins[ref] = true
			}
		}
	}
	return pins
}

func projectRecentBrews(recent []recentBrew, library catalog, did string) {
	for i := range recent {
		brew := &recent[i]
		brew.Fields = brewPreparation(brew.record)
		brew.Labels = map[string]string{}
		for j := range library {
			item := &library[j]
			ref, _ := brew.record[item.Kind+"Ref"].(string)
			if ref != "at://"+did+"/"+arabica.NSIDBase+"."+item.Kind+"/"+item.rkey {
				continue
			}
			if item.Kind != "bean" && item.Kind != "grinder" && item.Kind != "brewer" && item.Kind != "recipe" {
				continue
			}
			brew.Fields[item.Kind+"_rkey"] = item.ID
			brew.Labels[item.Kind] = item.Label
			if item.Kind == "bean" && item.RoasterLabel != "" {
				brew.Labels["roaster"] = item.RoasterLabel
			}
		}
	}
}

func brewPreparation(record map[string]any) map[string]any {
	out := map[string]any{}
	put := func(key string, value any) {
		if validateFields("brew", map[string]any{key: value}) == nil {
			out[key] = value
		}
	}
	for raw, field := range map[string]string{"method": "method", "grindSize": "grind_size"} {
		if value, ok := record[raw].(string); ok {
			put(field, compact(value, int(fields["brew"][field].Max)))
		}
	}
	for raw, field := range map[string]string{"coffeeAmount": "coffee_amount", "waterAmount": "water_amount", "temperature": "temperature", "timeSeconds": "time_seconds"} {
		if n, ok := recordNumber(record[raw]); ok {
			if raw == "temperature" {
				n /= 10
			}
			put(field, n)
		}
	}
	for block, mapping := range map[string]map[string]string{
		"espressoParams": {"yieldWeight": "yield_weight", "pressure": "pressure", "preInfusionSeconds": "pre_infusion_seconds"},
		"pouroverParams": {"bloomWater": "bloom_water", "bloomSeconds": "bloom_seconds", "drawdownSeconds": "drawdown_seconds", "bypassWater": "bypass_water", "filter": "filter"},
	} {
		params, _ := record[block].(map[string]any)
		prefix := "espresso_params."
		if block == "pouroverParams" {
			prefix = "pourover_params."
		}
		for raw, field := range mapping {
			if raw == "filter" {
				if value, ok := params[raw].(string); ok {
					put(prefix+field, compact(value, 100))
				}
				continue
			}
			if n, ok := recordNumber(params[raw]); ok {
				if block == "espressoParams" && (raw == "yieldWeight" || raw == "pressure") {
					n /= 10
				}
				put(prefix+field, n)
			}
		}
	}
	if pours := recipeDefaults(map[string]any{"pours": record["pours"]})["pours"]; pours != nil {
		put("pours", pours)
	}
	return out
}

func groundRecentBrew(recent []recentBrew, library catalog, id string) bool {
	if id == "" {
		return true
	}
	for _, brew := range recent {
		if brew.ID != id {
			continue
		}
		for i := range library {
			item := &library[i]
			if brew.Fields[item.Kind+"_rkey"] == item.ID {
				item.recentGround = true
			}
		}
		return true
	}
	return false
}
