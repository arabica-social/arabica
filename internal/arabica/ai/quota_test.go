package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	provider "tangled.org/arabica.social/arabica/internal/ai"
	"tangled.org/arabica.social/arabica/internal/records"
)

type fakeUsage struct {
	calls  int
	did    string
	day    string
	chars  int64
	limit  int64
	used   int64
	err    error
	purged []string
}

func (f *fakeUsage) Consume(_ context.Context, did, day string, chars, limit int64) (int64, bool, error) {
	f.calls++
	f.did, f.day, f.chars, f.limit = did, day, chars, limit
	if f.err != nil {
		return 0, false, f.err
	}
	if f.used+chars > limit {
		return f.used, false, nil
	}
	f.used += chars
	return f.used, true, nil
}

func (f *fakeUsage) PurgeBefore(_ context.Context, day string) error {
	f.purged = append(f.purged, day)
	return nil
}

func quotaService(t *testing.T, usage UsageStore, chars int64) *Service {
	t.Helper()
	service := NewService(outputClient(validOutput))
	service.SetUsageStore(usage, chars, nil)
	return service
}

func TestDailyQuotaChargesAndRefuses(t *testing.T) {
	usage := &fakeUsage{}
	service := quotaService(t, usage, 70)
	req := validRequest()
	cost := requestChars(req)
	if cost != int64(len("18 grams coffee with Own brewer")) {
		t.Fatalf("unexpected cost %d", cost)
	}
	for i := 0; i < 2; i++ {
		if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), req); err != nil {
			t.Fatal(err)
		}
	}
	if usage.calls != 2 || usage.did != ownDID || usage.limit != 70 || usage.chars != cost || usage.day == "" {
		t.Fatalf("usage not recorded: %+v", usage)
	}
	_, err := service.Chat(context.Background(), ownDID, ownLibrary(), req)
	if !errors.Is(err, ErrDailyLimit) {
		t.Fatalf("expected daily limit, got %v", err)
	}
	if usage.used != 2*cost {
		t.Fatalf("refused request was charged: %+v", usage)
	}
}

func TestDailyQuotaExemptsAdminAndListedDID(t *testing.T) {
	for name, configure := range map[string]func(*quota){
		"admin":  func(q *quota) { q.isAdmin = func(did string) bool { return did == ownDID } },
		"listed": func(q *quota) { q.unlimited = map[string]bool{ownDID: true} },
	} {
		t.Run(name, func(t *testing.T) {
			usage := &fakeUsage{}
			service := quotaService(t, usage, 1)
			configure(service.quota)
			for i := 0; i < 2; i++ {
				if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); err != nil {
					t.Fatal(err)
				}
			}
			if usage.calls != 0 {
				t.Fatalf("exempt account charged: %+v", usage)
			}
		})
	}
}

func TestDailyQuotaFailsClosedOnStoreError(t *testing.T) {
	usage := &fakeUsage{err: errors.New("database is locked")}
	service := quotaService(t, usage, 100)
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); !errors.Is(err, ErrQuota) {
		t.Fatalf("expected usage failure, got %v", err)
	}
}

func TestDailyQuotaZeroDisablesAccounting(t *testing.T) {
	usage := &fakeUsage{}
	service := quotaService(t, usage, 0)
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); err != nil {
		t.Fatal(err)
	}
	if usage.calls != 0 {
		t.Fatalf("disabled quota recorded usage: %+v", usage)
	}
}

func TestDailyQuotaNotChargedBeforeProviderCall(t *testing.T) {
	usage := &fakeUsage{}
	service := quotaService(t, usage, 100)
	broken := fakeLibrary{did: ownDID, fetch: func(context.Context, string) ([]records.RawRecord, error) {
		return nil, errors.New("pds unavailable")
	}}
	if _, err := service.Chat(context.Background(), ownDID, broken, validRequest()); !errors.Is(err, ErrLibrary) {
		t.Fatalf("expected library failure, got %v", err)
	}
	if usage.calls != 0 {
		t.Fatalf("failed library load was charged: %+v", usage)
	}
}

func TestDailyQuotaPaysPerImageAndHistory(t *testing.T) {
	req := Request{Messages: []Message{
		{Role: "user", Content: "abc"},
		{Role: "assistant", Content: "de"},
		{Role: "user", Content: "f", Images: []Image{{MIMEType: "image/png", Data: "x"}, {MIMEType: "image/png", Data: "y"}}},
	}}
	if got, want := requestChars(req), int64(3+2+1+2*imageChars); got != want {
		t.Fatalf("requestChars = %d, want %d", got, want)
	}
	usage := &fakeUsage{}
	service := NewService(fakeCompleter(func(context.Context, []provider.Message, []provider.Tool, json.RawMessage) (provider.Completion, error) {
		return provider.Completion{}, errors.New("not reached")
	}))
	service.SetUsageStore(usage, 100, nil)
	before := usage.calls
	if _, err := service.Chat(context.Background(), ownDID, ownLibrary(), validRequest()); err == nil {
		t.Fatal("provider failure ignored")
	}
	if usage.calls != before+1 {
		t.Fatalf("provider-bound request not charged: %+v", usage)
	}
}

func TestDailyQuotaReadsEnvironment(t *testing.T) {
	t.Setenv("ARABICA_AI_DAILY_CHARS", "250")
	t.Setenv("ARABICA_AI_UNLIMITED_DIDS", "did:plc:abcdefghijklmnopqrstuvwx, did:plc:zyxwvutsrqponmlkjihgfedc")
	q, err := dailyQuota()
	if err != nil || q.chars != 250 || len(q.unlimited) != 2 {
		t.Fatalf("env: %+v %v", q, err)
	}
	if !q.exempt(ownDID) || q.exempt("did:plc:someoneelse") {
		t.Fatal("unlimited list not applied")
	}
	t.Setenv("ARABICA_AI_DAILY_CHARS", "0")
	if q, err := dailyQuota(); err != nil || q.chars != 0 {
		t.Fatalf("disabled: %+v %v", q, err)
	}
	for _, bad := range []string{"-1", "many", "1e9x"} {
		t.Setenv("ARABICA_AI_DAILY_CHARS", bad)
		if _, err := dailyQuota(); err == nil {
			t.Fatalf("accepted %q", bad)
		}
	}
	t.Setenv("ARABICA_AI_DAILY_CHARS", "")
	t.Setenv("ARABICA_AI_UNLIMITED_DIDS", "not-a-did")
	if _, err := dailyQuota(); err == nil {
		t.Fatal("accepted an invalid unlimited DID")
	}
	if got, err := dailyQuota(); err == nil && got.chars != DefaultDailyChars {
		t.Fatalf("default %d", got.chars)
	}
}
