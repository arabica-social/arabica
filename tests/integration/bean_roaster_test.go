//go:build integration

package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTP_BeanCreateNamedRoaster(t *testing.T) {
	h := StartHarness(t, nil)
	create := func(bean, name string) string {
		res := h.PostJSON("/api/beans", map[string]string{"name": bean, "origin": "Kenya", "new_roaster_name": name})
		body := ReadBody(t, res)
		require.Equal(t, http.StatusOK, res.StatusCode, body)
		var saved struct {
			RoasterKey string `json:"roaster_rkey"`
			Roaster    struct {
				Name string `json:"name"`
				Key  string `json:"rkey"`
			} `json:"roaster"`
		}
		require.NoError(t, json.Unmarshal([]byte(body), &saved))
		require.NotEmpty(t, saved.RoasterKey)
		require.Equal(t, saved.RoasterKey, saved.Roaster.Key)
		require.Equal(t, "North Star", saved.Roaster.Name)
		return saved.RoasterKey
	}
	first := create("Pink Bourbon", "North Star")
	require.Equal(t, first, create("Caturra", " north   star "))

	mustRKey(t, h.PostForm("/api/roasters", form("name", "North Star", "location", "Oslo")), "roaster")
	res := h.PostJSON("/api/beans", map[string]string{"name": "Avoid ambiguity", "origin": "Kenya", "new_roaster_name": "North Star"})
	body := ReadBody(t, res)
	require.Equal(t, http.StatusConflict, res.StatusCode, body)
	require.Contains(t, body, "ambiguous_roaster")
}
