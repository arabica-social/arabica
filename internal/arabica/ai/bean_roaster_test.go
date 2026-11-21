package ai

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

func TestBeanDraftRoaster(t *testing.T) {
	for _, tt := range []struct {
		name, message      string
		saved              []string
		proposal, expected map[string]any
		choices            int
	}{
		{"opaque selection", "Add Pink Bourbon from North Star", []string{"North Star"}, map[string]any{"roaster_rkey": "roaster-1"}, map[string]any{"roaster_rkey": "saved-1"}, 0},
		{"reuse named selection", "Add Pink Bourbon from North Star", []string{"North Star"}, map[string]any{"new_roaster_name": "north   star"}, map[string]any{"roaster_rkey": "saved-1"}, 0},
		{"new named roaster", "Add Pink Bourbon from North Star", nil, map[string]any{"new_roaster_name": " North  Star "}, map[string]any{"new_roaster_name": "North Star"}, 0},
		{"ambiguous name is not a create", "Add Pink Bourbon from North Star", []string{"North Star", "north star"}, map[string]any{"new_roaster_name": "North Star"}, map[string]any{}, 0},
		{"unmentioned name needs confirmation to select a saved roaster", "Add this coffee", []string{"North Star"}, map[string]any{"new_roaster_name": "North Star"}, map[string]any{}, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			library := fakeLibrary{did: ownDID, fetch: func(_ context.Context, nsid string) ([]records.RawRecord, error) {
				var rows []records.RawRecord
				if nsid == arabica.NSIDRoaster {
					for i, name := range tt.saved {
						rows = append(rows, records.RawRecord{RKey: []string{"saved-1", "saved-2"}[i], Record: map[string]any{"name": name}})
					}
				}
				return rows, nil
			}}
			req := validRequest()
			req.Messages[0].Content = tt.message
			tt.proposal["name"] = "Pink Bourbon"
			tt.expected["name"] = "Pink Bourbon"
			body, err := json.Marshal(map[string]any{"in_scope": true, "reply": "Bean details", "kind": "bean", "patch": tt.proposal, "choices": []any{}})
			require.NoError(t, err)
			result, err := NewService(outputClient(string(body))).Chat(context.Background(), ownDID, library, req)
			require.NoError(t, err)
			require.Equal(t, tt.expected, result.Patch)
			require.Len(t, result.Choices, tt.choices)
		})
	}
}
