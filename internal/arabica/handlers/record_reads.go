package coffeehandlers

import (
	"context"
	"fmt"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

func listGrinders(ctx context.Context, store records.Store) ([]*arabica.Grinder, error) {
	return listArabicaRecords(ctx, store, arabica.NSIDGrinder, arabica.RecordToGrinder)
}

func getGrinder(ctx context.Context, store records.Store, rkey string) (*arabica.Grinder, error) {
	return getArabicaRecord(ctx, store, arabica.NSIDGrinder, rkey, arabica.RecordToGrinder)
}

func listBrewers(ctx context.Context, store records.Store) ([]*arabica.Brewer, error) {
	return listArabicaRecords(ctx, store, arabica.NSIDBrewer, arabica.RecordToBrewer)
}

func getBrewer(ctx context.Context, store records.Store, rkey string) (*arabica.Brewer, error) {
	return getArabicaRecord(ctx, store, arabica.NSIDBrewer, rkey, arabica.RecordToBrewer)
}

func listArabicaRecords[T any](ctx context.Context, store records.Store, nsid string, decode func(map[string]any, string) (*T, error)) ([]*T, error) {
	raw, err := store.FetchAllRecords(ctx, nsid)
	if err != nil {
		return nil, err
	}
	out := make([]*T, 0, len(raw))
	for _, r := range raw {
		m, err := decode(r.Record, r.URI)
		if err != nil {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func getArabicaRecord[T any](ctx context.Context, store records.Store, nsid, rkey string, decode func(map[string]any, string) (*T, error)) (*T, error) {
	rec, uri, _, err := store.FetchRecord(ctx, nsid, rkey)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, fmt.Errorf("%s %s not found", nsid, rkey)
	}
	return decode(rec, uri)
}
