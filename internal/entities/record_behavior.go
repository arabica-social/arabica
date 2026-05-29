package entities

import (
	"fmt"

	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// RecordBehavior holds record-specific behavior for a descriptor. It is split
// from Descriptor so domain identity metadata does not also own codecs and
// typed model accessors.
type RecordBehavior struct {
	// GetField extracts one named string field from a typed model pointer for
	// form prefill. Returns ("", false) if entity is nil or field is unknown.
	GetField func(entity any, field string) (string, bool)

	// RecordToModel converts a raw record map (as fetched from PDS or witness
	// cache) into the typed model for this entity. Returns the model as any;
	// callers type-assert. nil callback means the entity does not appear in
	// the feed pipeline.
	RecordToModel func(record map[string]any, uri string) (any, error)

	// RKey returns the record key of a typed record. The argument is the
	// concrete record pointer (e.g. *arabica.Bean) typed as any to avoid
	// import cycles. Returns "" if the assertion fails or the record is nil.
	RKey func(record any) string

	// DisplayTitle returns a human-readable title for a typed record.
	// Empty string means "use the descriptor's DisplayName as fallback."
	DisplayTitle func(record any) string

	// ResolveRefs hydrates cross-entity references on a typed model using
	// records already pulled from the firehose index. nil means the entity has
	// no cross-record references to resolve.
	ResolveRefs func(model any, recordData map[string]any, lookup func(refURI string) (map[string]any, bool))
}

var behaviorRegistry = map[lexicons.RecordType]*RecordBehavior{}

func RegisterRecordBehavior(rt lexicons.RecordType, b *RecordBehavior) {
	if _, ok := behaviorRegistry[rt]; ok {
		panic(fmt.Sprintf("entities: duplicate record behavior for %s", rt))
	}
	behaviorRegistry[rt] = b
}

func Behavior(rt lexicons.RecordType) *RecordBehavior {
	return behaviorRegistry[rt]
}

func BehaviorByNSID(nsid string) *RecordBehavior {
	d := GetByNSID(nsid)
	if d == nil {
		return nil
	}
	return Behavior(d.Type)
}
