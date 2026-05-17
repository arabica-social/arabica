package oolong

import "time"

type Cafe struct {
	RKey        string    `json:"rkey"`
	Name        string    `json:"name"`
	Location    string    `json:"location"`
	Address     string    `json:"address"`
	Website     string    `json:"website"`
	Description string    `json:"description"`
	VendorRKey  string    `json:"vendor_rkey,omitempty"`
	SourceRef   string    `json:"source_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`

	// Joined data for display
	Vendor *Vendor `json:"vendor,omitempty"`
}

type CreateCafeRequest struct {
	Name        string `json:"name"`
	Location    string `json:"location"`
	Address     string `json:"address"`
	Website     string `json:"website"`
	Description string `json:"description"`
	VendorRKey  string `json:"vendor_rkey,omitempty"`
	SourceRef   string `json:"source_ref,omitempty"`
}

type UpdateCafeRequest CreateCafeRequest

func (r *CreateCafeRequest) Validate() error {
	if r.Name == "" {
		return ErrNameRequired
	}
	if len(r.Name) > MaxNameLength {
		return ErrNameTooLong
	}
	if len(r.Location) > MaxLocationLength {
		return ErrLocationTooLong
	}
	if len(r.Address) > MaxAddressLength {
		return ErrFieldTooLong
	}
	if len(r.Website) > MaxWebsiteLength {
		return ErrWebsiteTooLong
	}
	if len(r.Description) > MaxDescriptionLength {
		return ErrDescTooLong
	}
	return nil
}

func (r *UpdateCafeRequest) Validate() error {
	c := CreateCafeRequest(*r)
	return c.Validate()
}
