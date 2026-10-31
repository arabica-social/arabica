package ai

import (
	"context"
	"fmt"
	"tangled.org/arabica.social/arabica/internal/records"
	"testing"
)

func TestRoasterBeanGroundingContext(t *testing.T) {
	rows := map[string][]records.RawRecord{
		"roaster": {contextRow("recluse", "Recluse", nil), contextRow("duplicate", "Recluse", nil)},
		"bean":    {contextRow("open", "Ethiopia", map[string]any{"roasterRef": contextRef("roaster", "recluse")}), contextRow("closed", "Colombia", map[string]any{"closed": true, "roasterRef": contextRef("roaster", "recluse")}), contextRow("unknown", "Unknown", map[string]any{"closed": "false", "roasterRef": contextRef("roaster", "recluse")}), contextRow("foreign", "Foreign", map[string]any{"roasterRef": "at://did:plc:foreign/social.arabica.alpha.roaster/recluse"}), contextRow("missing", "Missing", nil)},
	}
	load := func() catalog {
		t.Helper()
		c, err := loadCatalog(context.Background(), contextLibrary(rows), Draft{}, "brew from Recluse")
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	c := load()
	open := findContext(t, c, "bean", "open")
	if open.Closed == nil || *open.Closed || open.OpenRoasterMatches != 1 || !c.explicitlyNamed("bean_rkey", "open", "brew from Recluse") {
		t.Fatalf("single legacy-open bean %+v", open)
	}
	for _, key := range []string{"closed", "unknown", "foreign", "missing"} {
		if c.explicitlyNamed("bean_rkey", key, "brew from Recluse") {
			t.Fatalf("auto selected %s", key)
		}
	}
	if got := c.deriveLabel(map[string]any{"bean_rkey": "open"}); got != "Ethiopia · Recluse" {
		t.Fatal(got)
	}
	rows["bean"] = append(rows["bean"], contextRow("second", "Kenya", map[string]any{"roasterRef": contextRef("roaster", "duplicate")}))
	c = load()
	if c.explicitlyNamed("bean_rkey", "open", "brew from Recluse") || findContext(t, c, "bean", "open").OpenRoasterMatches != 2 {
		t.Fatal("duplicate roaster labels not counted together")
	}
}
func TestRoasterAmbiguityBeyondCap(t *testing.T) {
	rows := map[string][]records.RawRecord{"roaster": {contextRow("r", "Recluse", nil)}}
	for i := 0; i < 105; i++ {
		rows["bean"] = append(rows["bean"], contextRow(fmt.Sprintf("bean%d", i), fmt.Sprintf("Bean %d", i), map[string]any{"roasterRef": contextRef("roaster", "r")}))
	}
	c, err := loadCatalog(context.Background(), contextLibrary(rows), Draft{Fields: map[string]any{"bean_rkey": "bean104"}}, "Recluse")
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, item := range c {
		if item.Kind == "bean" {
			count++
			if item.OpenRoasterMatches != 105 {
				t.Fatal(item.OpenRoasterMatches)
			}
		}
	}
	if count != 100 {
		t.Fatal(count)
	}
	if c.explicitlyNamed("bean_rkey", "bean104", "Recluse") {
		t.Fatal("ambiguity outside cap ignored")
	}
	findContext(t, c, "bean", "bean104")
}

func TestRoasterNameIsNotIndependentBeanNameEvidence(t *testing.T) {
	closed, open := true, false
	for _, item := range []libraryItem{
		{Kind: "bean", rkey: "bag", Label: "Recluse", RoasterLabel: "Recluse", Closed: &closed, uniqueLabel: true, OpenRoasterMatches: 1},
		{Kind: "bean", rkey: "bag", Label: "Recluse", RoasterLabel: "Recluse", Closed: &open, uniqueLabel: true, OpenRoasterMatches: 2},
	} {
		if (catalog{item}).explicitlyNamed("bean_rkey", "bag", "brew from Recluse") {
			t.Fatalf("roaster-only auto-selection %+v", item)
		}
	}
}

func TestClosedBeanChoiceLabel(t *testing.T) {
	closed, open := true, false
	for _, tc := range []struct {
		status *bool
		want   string
	}{{&closed, "Ethiopia · Recluse (closed)"}, {&open, "Ethiopia · Recluse"}, {nil, "Ethiopia · Recluse"}} {
		c := catalog{{Kind: "bean", rkey: "bag", Label: "Ethiopia", RoasterLabel: "Recluse", Closed: tc.status}}
		if got := c.deriveLabel(map[string]any{"bean_rkey": "bag"}); got != tc.want {
			t.Fatalf("label=%q want %q", got, tc.want)
		}
	}
}
