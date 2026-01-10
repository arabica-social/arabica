package bff

import (
	"testing"

	"arabica/internal/models"
)

func TestFormatTemp(t *testing.T) {
	tests := []struct {
		name     string
		temp     float64
		expected string
	}{
		{"zero returns N/A", 0, "N/A"},
		{"celsius range", 93.5, "93.5°C"},
		{"celsius whole number", 90.0, "90.0°C"},
		{"celsius at 100", 100.0, "100.0°C"},
		{"fahrenheit range", 200.0, "200.0°F"},
		{"fahrenheit at 212", 212.0, "212.0°F"},
		{"low temp celsius", 20.5, "20.5°C"},
		{"just over 100 is fahrenheit", 100.1, "100.1°F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTemp(tt.temp)
			if got != tt.expected {
				t.Errorf("FormatTemp(%v) = %q, want %q", tt.temp, got, tt.expected)
			}
		})
	}
}

func TestFormatTempValue(t *testing.T) {
	tests := []struct {
		name     string
		temp     float64
		expected string
	}{
		{"zero", 0, "0.0"},
		{"whole number", 93.0, "93.0"},
		{"decimal", 93.5, "93.5"},
		{"high precision rounds", 93.55, "93.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTempValue(tt.temp)
			if got != tt.expected {
				t.Errorf("FormatTempValue(%v) = %q, want %q", tt.temp, got, tt.expected)
			}
		})
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		name     string
		seconds  int
		expected string
	}{
		{"zero returns N/A", 0, "N/A"},
		{"seconds only", 30, "30s"},
		{"exactly one minute", 60, "1m"},
		{"minutes and seconds", 90, "1m 30s"},
		{"multiple minutes", 180, "3m"},
		{"multiple minutes and seconds", 185, "3m 5s"},
		{"large time", 3661, "61m 1s"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatTime(tt.seconds)
			if got != tt.expected {
				t.Errorf("FormatTime(%v) = %q, want %q", tt.seconds, got, tt.expected)
			}
		})
	}
}

func TestFormatRating(t *testing.T) {
	tests := []struct {
		name     string
		rating   int
		expected string
	}{
		{"zero returns N/A", 0, "N/A"},
		{"rating 1", 1, "1/10"},
		{"rating 5", 5, "5/10"},
		{"rating 10", 10, "10/10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatRating(tt.rating)
			if got != tt.expected {
				t.Errorf("FormatRating(%v) = %q, want %q", tt.rating, got, tt.expected)
			}
		})
	}
}

func TestFormatID(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		expected string
	}{
		{"zero", 0, "0"},
		{"positive", 123, "123"},
		{"large number", 99999, "99999"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatID(tt.id)
			if got != tt.expected {
				t.Errorf("FormatID(%v) = %q, want %q", tt.id, got, tt.expected)
			}
		})
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		name     string
		val      int
		expected string
	}{
		{"zero", 0, "0"},
		{"positive", 42, "42"},
		{"negative", -5, "-5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatInt(tt.val)
			if got != tt.expected {
				t.Errorf("FormatInt(%v) = %q, want %q", tt.val, got, tt.expected)
			}
		})
	}
}

func TestFormatRoasterID(t *testing.T) {
	t.Run("nil returns null", func(t *testing.T) {
		got := FormatRoasterID(nil)
		if got != "null" {
			t.Errorf("FormatRoasterID(nil) = %q, want %q", got, "null")
		}
	})

	t.Run("valid pointer", func(t *testing.T) {
		id := 123
		got := FormatRoasterID(&id)
		if got != "123" {
			t.Errorf("FormatRoasterID(&123) = %q, want %q", got, "123")
		}
	})

	t.Run("zero pointer", func(t *testing.T) {
		id := 0
		got := FormatRoasterID(&id)
		if got != "0" {
			t.Errorf("FormatRoasterID(&0) = %q, want %q", got, "0")
		}
	})
}

func TestPoursToJSON(t *testing.T) {
	tests := []struct {
		name     string
		pours    []*models.Pour
		expected string
	}{
		{
			name:     "empty pours",
			pours:    []*models.Pour{},
			expected: "[]",
		},
		{
			name:     "nil pours",
			pours:    nil,
			expected: "[]",
		},
		{
			name: "single pour",
			pours: []*models.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
			},
			expected: `[{"water":50,"time":30}]`,
		},
		{
			name: "multiple pours",
			pours: []*models.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
				{WaterAmount: 100, TimeSeconds: 60},
				{WaterAmount: 150, TimeSeconds: 90},
			},
			expected: `[{"water":50,"time":30},{"water":100,"time":60},{"water":150,"time":90}]`,
		},
		{
			name: "zero values",
			pours: []*models.Pour{
				{WaterAmount: 0, TimeSeconds: 0},
			},
			expected: `[{"water":0,"time":0}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PoursToJSON(tt.pours)
			if got != tt.expected {
				t.Errorf("PoursToJSON() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestPtr(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		p := Ptr(42)
		if *p != 42 {
			t.Errorf("Ptr(42) = %v, want 42", *p)
		}
	})

	t.Run("string", func(t *testing.T) {
		p := Ptr("hello")
		if *p != "hello" {
			t.Errorf("Ptr(\"hello\") = %v, want \"hello\"", *p)
		}
	})

	t.Run("zero value", func(t *testing.T) {
		p := Ptr(0)
		if *p != 0 {
			t.Errorf("Ptr(0) = %v, want 0", *p)
		}
	})
}

func TestPtrEquals(t *testing.T) {
	t.Run("nil pointer returns false", func(t *testing.T) {
		var p *int = nil
		if PtrEquals(p, 42) {
			t.Error("PtrEquals(nil, 42) should be false")
		}
	})

	t.Run("matching value returns true", func(t *testing.T) {
		val := 42
		if !PtrEquals(&val, 42) {
			t.Error("PtrEquals(&42, 42) should be true")
		}
	})

	t.Run("non-matching value returns false", func(t *testing.T) {
		val := 42
		if PtrEquals(&val, 99) {
			t.Error("PtrEquals(&42, 99) should be false")
		}
	})

	t.Run("string comparison", func(t *testing.T) {
		s := "hello"
		if !PtrEquals(&s, "hello") {
			t.Error("PtrEquals(&\"hello\", \"hello\") should be true")
		}
		if PtrEquals(&s, "world") {
			t.Error("PtrEquals(&\"hello\", \"world\") should be false")
		}
	})
}

func TestPtrValue(t *testing.T) {
	t.Run("nil int returns zero", func(t *testing.T) {
		var p *int = nil
		if PtrValue(p) != 0 {
			t.Errorf("PtrValue(nil) = %v, want 0", PtrValue(p))
		}
	})

	t.Run("valid int returns value", func(t *testing.T) {
		val := 42
		if PtrValue(&val) != 42 {
			t.Errorf("PtrValue(&42) = %v, want 42", PtrValue(&val))
		}
	})

	t.Run("nil string returns empty", func(t *testing.T) {
		var p *string = nil
		if PtrValue(p) != "" {
			t.Errorf("PtrValue(nil string) = %v, want \"\"", PtrValue(p))
		}
	})

	t.Run("valid string returns value", func(t *testing.T) {
		s := "hello"
		if PtrValue(&s) != "hello" {
			t.Errorf("PtrValue(&\"hello\") = %v, want \"hello\"", PtrValue(&s))
		}
	})
}
