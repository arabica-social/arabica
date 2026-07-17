package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
)

func TestServerAppUsesExpectedDefaults(t *testing.T) {
	run := appRun{
		app:                arabicaapp.New(),
		defaultPort:        "18910",
		defaultMetricsPort: "9101",
	}

	assert.Equal(t, "arabica", run.app.Name)
	assert.Equal(t, "18910", run.defaultPort)
	assert.Equal(t, "9101", run.defaultMetricsPort)
}

func TestServerAppUsesArabicaNSIDBase(t *testing.T) {
	run := appRun{app: arabicaapp.New()}
	assert.Equal(t, "social.arabica.alpha", run.app.NSIDBase)
}
