package ai

import (
	"context"
	"strings"
	"testing"

	"tangled.org/arabica.social/arabica/internal/records"
)

func TestExactMentionBoundaries(t *testing.T) {
	for _, tt := range []struct {
		message, label string
		want           bool
	}{
		{"I used OWN   BREWER today", "Own brewer", true},
		{"I used my V60.", "v60", true},
		{"I used a V600", "V60", false},
		{"prefixOwn brewer suffix", "Own brewer", false},
		{"Own brewers are nice", "Own brewer", false},
		{"anything", " ", false},
		{"豆 コーヒー", "コーヒー", true},
	} {
		if got := exactMention(tt.message, tt.label); got != tt.want {
			t.Errorf("message=%q label=%q got=%v", tt.message, tt.label, got)
		}
	}
}

func TestSoleUnmentionedBeanDefaultsButEquipmentRequiresChoice(t *testing.T) {
	req := validRequest()
	req.Messages[0].Content = "18 grams of coffee"
	response, err := NewService(outputClient(validOutput)).Chat(context.Background(), ownDID, ownLibrary(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.Patch["brewer_rkey"]; ok || len(response.Choices) != 1 || response.Patch["bean_rkey"] != "real-bean" {
		t.Fatalf("unmentioned sole item autofilled %+v", response)
	}
}

func TestDuplicateLabelsBeyondCatalogPreventAutofill(t *testing.T) {
	lib := ownLibrary()
	base := lib.fetch
	lib.fetch = func(ctx context.Context, nsid string) ([]records.RawRecord, error) {
		if !strings.HasSuffix(nsid, ".brewer") {
			return base(ctx, nsid)
		}
		result := make([]records.RawRecord, 101)
		for i := range result {
			result[i] = records.RawRecord{RKey: strings.Repeat("x", i+1), Record: map[string]any{"name": "Other"}}
		}
		result[0] = records.RawRecord{RKey: "real-brewer", Record: map[string]any{"name": "Own brewer", "brewerType": "pourover"}}
		result[100] = records.RawRecord{RKey: "duplicate-brewer", Record: map[string]any{"name": "  OWN  BREWER "}}
		return result, nil
	}
	response, err := NewService(outputClient(validOutput)).Chat(context.Background(), ownDID, lib, validRequest())
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.Patch["brewer_rkey"]; ok {
		t.Fatalf("duplicate outside cap auto-selected %+v", response)
	}
}

func TestUnconfirmedBrewerChoiceCarriesDependentDetails(t *testing.T) {
	req := validRequest()
	req.Messages[0].Content = "18 grams and a new brewer with paper filter"
	output := strings.Replace(validOutput, `"coffee_amount":18`, `"coffee_amount":18,"pourover_params.filter":"paper"`, 1)
	response, err := NewService(outputClient(output)).Chat(context.Background(), ownDID, ownLibrary(), req)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.Patch["pourover_params.filter"]; ok {
		t.Fatal("dependent equipment detail applied before selection")
	}
	if response.Patch["coffee_amount"] != float64(18) || response.Choices[len(response.Choices)-1].Patch["pourover_params.filter"] != "paper" {
		t.Fatalf("dependent values lost %+v", response)
	}
}
