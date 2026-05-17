package oolong

import "time"

type Vendor struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	Website     string    `json:"website"`
	Description string    `json:"description"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateVendorRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Website     string `json:"website"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateVendorRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Website     string `json:"website"`
	Description string `json:"description"`
	SourceRef   string `json:"source_ref,omitempty"`
}

func (r *CreateVendorRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Location) > MaxLocationLength {
		return ErrLocationTooLong
	}
	if len(r.Website) > MaxWebsiteLength {
		return ErrWebsiteTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	return nil
}

func (r *UpdateVendorRequest) Validate() error {
	c := CreateVendorRequest(*r)
	return c.Validate()
}
