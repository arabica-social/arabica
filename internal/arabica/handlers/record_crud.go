package coffeehandlers

import (
	"net/http"
	"time"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

func decodeRoasterCreateForm(r *http.Request, req *arabica.CreateRoasterRequest) error {
	*req = arabica.CreateRoasterRequest{
		Name: r.FormValue("name"), Location: r.FormValue("location"),
		Website: r.FormValue("website"), SourceRef: r.FormValue("source_ref"),
	}
	return nil
}

func decodeRoasterUpdateForm(r *http.Request, req *arabica.UpdateRoasterRequest) error {
	*req = arabica.UpdateRoasterRequest{
		Name: r.FormValue("name"), Location: r.FormValue("location"),
		Website: r.FormValue("website"), SourceRef: r.FormValue("source_ref"),
	}
	return nil
}

func roasterFromCreate(req *arabica.CreateRoasterRequest, createdAt time.Time) *arabica.Roaster {
	return &arabica.Roaster{Name: req.Name, Location: req.Location, Website: req.Website, SourceRef: req.SourceRef, CreatedAt: createdAt}
}

func roasterFromUpdate(req *arabica.UpdateRoasterRequest, createdAt time.Time) *arabica.Roaster {
	return &arabica.Roaster{Name: req.Name, Location: req.Location, Website: req.Website, SourceRef: req.SourceRef, CreatedAt: createdAt}
}

func grinderFromCreate(req *arabica.CreateGrinderRequest, createdAt time.Time) *arabica.Grinder {
	return &arabica.Grinder{
		Name: req.Name, GrinderType: req.GrinderType, BurrType: req.BurrType,
		Notes: req.Notes, Link: req.Link, SourceRef: req.SourceRef, CreatedAt: createdAt,
	}
}

func grinderFromUpdate(req *arabica.UpdateGrinderRequest, createdAt time.Time) *arabica.Grinder {
	return &arabica.Grinder{
		Name: req.Name, GrinderType: req.GrinderType, BurrType: req.BurrType,
		Notes: req.Notes, Link: req.Link, SourceRef: req.SourceRef, CreatedAt: createdAt,
	}
}

func brewerFromCreate(req *arabica.CreateBrewerRequest, createdAt time.Time) *arabica.Brewer {
	return &arabica.Brewer{
		Name: req.Name, BrewerType: req.BrewerType, Description: req.Description,
		Link: req.Link, SourceRef: req.SourceRef, CreatedAt: createdAt,
	}
}

func brewerFromUpdate(req *arabica.UpdateBrewerRequest, createdAt time.Time) *arabica.Brewer {
	return &arabica.Brewer{
		Name: req.Name, BrewerType: req.BrewerType, Description: req.Description,
		Link: req.Link, SourceRef: req.SourceRef, CreatedAt: createdAt,
	}
}
