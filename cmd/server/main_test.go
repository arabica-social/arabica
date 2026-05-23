package main

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	oolongapp "tangled.org/arabica.social/arabica/internal/oolong/app"
)

func TestServerAppsUseSeparateDefaults(t *testing.T) {
	runs := []appRun{
		{app: arabicaapp.New(), defaultPort: "18910", defaultMetricsPort: "9101"},
		{app: oolongapp.New(), defaultPort: "18920", defaultMetricsPort: "9102"},
	}

	got := map[string]appRun{}
	for _, run := range runs {
		got[run.app.Name] = run
	}

	assert.Equal(t, "18910", got["arabica"].defaultPort)
	assert.Equal(t, "9101", got["arabica"].defaultMetricsPort)
	assert.Equal(t, "18920", got["oolong"].defaultPort)
	assert.Equal(t, "9102", got["oolong"].defaultMetricsPort)
}

func TestServerAppsUseDistinctNSIDBases(t *testing.T) {
	runs := []appRun{
		{app: arabicaapp.New(), defaultPort: "18910", defaultMetricsPort: "9101"},
		{app: oolongapp.New(), defaultPort: "18920", defaultMetricsPort: "9102"},
	}

	bases := make([]string, 0, len(runs))
	for _, run := range runs {
		bases = append(bases, run.app.NSIDBase)
	}
	sort.Strings(bases)

	assert.Equal(t, []string{"social.arabica.alpha", "social.oolong.alpha"}, bases)
}
