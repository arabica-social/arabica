package coffeepages

import (
	"encoding/json"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

// PoursToJSON serializes a slice of pours to JSON for use in the brew form.
func PoursToJSON(pours []*arabica.Pour) string {
	if len(pours) == 0 {
		return "[]"
	}

	type pourData struct {
		Water int `json:"water"`
		Time  int `json:"time"`
	}

	data := make([]pourData, len(pours))
	for i, p := range pours {
		data[i] = pourData{
			Water: p.WaterAmount,
			Time:  p.TimeSeconds,
		}
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "[]"
	}

	return string(jsonBytes)
}
