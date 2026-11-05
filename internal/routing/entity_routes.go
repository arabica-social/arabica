package routing

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

type entityRouteBundle struct {
	RecordType lexicons.RecordType

	Create http.HandlerFunc
	Update http.HandlerFunc
	Delete http.HandlerFunc

	JSONView http.HandlerFunc

	JSONBacklinks http.HandlerFunc

	OGImage http.HandlerFunc
}

func entityRouteBundles(h *handlers.Handler) []entityRouteBundle {
	return []entityRouteBundle{
		{
			RecordType:    lexicons.RecordTypeBean,
			Create:        h.HandleBeanCreate,
			Update:        h.HandleBeanUpdate,
			Delete:        h.HandleBeanDelete,
			JSONView:      h.HandleBeanViewJSON,
			JSONBacklinks: h.HandleBeanBacklinksJSON,
			OGImage:       h.HandleBeanOGImage,
		},
		{
			RecordType:    lexicons.RecordTypeRoaster,
			Create:        h.HandleRoasterCreate,
			Update:        h.HandleRoasterUpdate,
			Delete:        h.HandleRoasterDelete,
			JSONView:      h.HandleRoasterViewJSON,
			JSONBacklinks: h.HandleRoasterBacklinksJSON,
			OGImage:       h.HandleRoasterOGImage,
		},
		{
			RecordType:    lexicons.RecordTypeGrinder,
			Create:        h.HandleGrinderCreate,
			Update:        h.HandleGrinderUpdate,
			Delete:        h.HandleGrinderDelete,
			JSONView:      h.HandleGrinderViewJSON,
			JSONBacklinks: h.HandleGrinderBacklinksJSON,
			OGImage:       h.HandleGrinderOGImage,
		},
		{
			RecordType:    lexicons.RecordTypeBrewer,
			Create:        h.HandleBrewerCreate,
			Update:        h.HandleBrewerUpdate,
			Delete:        h.HandleBrewerDelete,
			JSONView:      h.HandleBrewerViewJSON,
			JSONBacklinks: h.HandleBrewerBacklinksJSON,
			OGImage:       h.HandleBrewerOGImage,
		},
	}
}
