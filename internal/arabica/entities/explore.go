package arabica

import (
	"tangled.org/arabica.social/arabica/internal/explore"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// init registers Arabica's explore registry factory so the firehose can build
// an explore index for coffee records without importing this package. Keyed by
// the app NSID base. Oolong registers no explore factory (it has no explore
// surface); explore.RegistryFor returns nil for it and the firehose treats
// explore as a no-op.
func init() {
	explore.Register(NSIDBase, func(nsids map[lexicons.RecordType]string) *explore.Registry {
		return explore.NewArabicaRegistry(nsids)
	})
}
