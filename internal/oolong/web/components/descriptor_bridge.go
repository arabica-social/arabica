package tea

import (
	"tangled.org/arabica.social/arabica/internal/lexicons"
	"tangled.org/arabica.social/arabica/internal/web/feedviews"

	// Ensure oolong descriptors are registered before we attach hooks.
	_ "tangled.org/arabica.social/arabica/internal/oolong/entities"
)

func FeedViews() feedviews.Registry {
	return feedviews.Registry{
		lexicons.RecordTypeOolongTea:     {Render: TeaFeedContent},
		lexicons.RecordTypeOolongVendor:  {Render: VendorFeedContent, Compact: true},
		lexicons.RecordTypeOolongVessel:  {Render: VesselFeedContent, Compact: true},
		lexicons.RecordTypeOolongInfuser: {Render: InfuserFeedContent, Compact: true},
		lexicons.RecordTypeOolongBrew:    {Render: BrewFeedContent},
	}
}
