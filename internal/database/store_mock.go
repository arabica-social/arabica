package database

import (
	"context"

	"arabica/internal/models"
)

// MockStore is a mock implementation of the Store interface for testing.
// Uses function fields to allow tests to inject custom behavior.
type MockStore struct {
	// Brew operations
	CreateBrewFunc       func(ctx context.Context, brew *models.CreateBrewRequest, userID int) (*models.Brew, error)
	GetBrewByRKeyFunc    func(ctx context.Context, rkey string) (*models.Brew, error)
	ListBrewsFunc        func(ctx context.Context, userID int) ([]*models.Brew, error)
	UpdateBrewByRKeyFunc func(ctx context.Context, rkey string, brew *models.CreateBrewRequest) error
	DeleteBrewByRKeyFunc func(ctx context.Context, rkey string) error

	// Bean operations
	CreateBeanFunc       func(ctx context.Context, bean *models.CreateBeanRequest) (*models.Bean, error)
	GetBeanByRKeyFunc    func(ctx context.Context, rkey string) (*models.Bean, error)
	ListBeansFunc        func(ctx context.Context) ([]*models.Bean, error)
	UpdateBeanByRKeyFunc func(ctx context.Context, rkey string, bean *models.UpdateBeanRequest) error
	DeleteBeanByRKeyFunc func(ctx context.Context, rkey string) error

	// Roaster operations
	CreateRoasterFunc       func(ctx context.Context, roaster *models.CreateRoasterRequest) (*models.Roaster, error)
	GetRoasterByRKeyFunc    func(ctx context.Context, rkey string) (*models.Roaster, error)
	ListRoastersFunc        func(ctx context.Context) ([]*models.Roaster, error)
	UpdateRoasterByRKeyFunc func(ctx context.Context, rkey string, roaster *models.UpdateRoasterRequest) error
	DeleteRoasterByRKeyFunc func(ctx context.Context, rkey string) error

	// Grinder operations
	CreateGrinderFunc       func(ctx context.Context, grinder *models.CreateGrinderRequest) (*models.Grinder, error)
	GetGrinderByRKeyFunc    func(ctx context.Context, rkey string) (*models.Grinder, error)
	ListGrindersFunc        func(ctx context.Context) ([]*models.Grinder, error)
	UpdateGrinderByRKeyFunc func(ctx context.Context, rkey string, grinder *models.UpdateGrinderRequest) error
	DeleteGrinderByRKeyFunc func(ctx context.Context, rkey string) error

	// Brewer operations
	CreateBrewerFunc       func(ctx context.Context, brewer *models.CreateBrewerRequest) (*models.Brewer, error)
	GetBrewerByRKeyFunc    func(ctx context.Context, rkey string) (*models.Brewer, error)
	ListBrewersFunc        func(ctx context.Context) ([]*models.Brewer, error)
	UpdateBrewerByRKeyFunc func(ctx context.Context, rkey string, brewer *models.UpdateBrewerRequest) error
	DeleteBrewerByRKeyFunc func(ctx context.Context, rkey string) error

	CloseFunc func() error
}

// CreateBrew calls the mock function or returns nil if not set
func (m *MockStore) CreateBrew(ctx context.Context, brew *models.CreateBrewRequest, userID int) (*models.Brew, error) {
	if m.CreateBrewFunc != nil {
		return m.CreateBrewFunc(ctx, brew, userID)
	}
	return nil, nil
}

// GetBrewByRKey calls the mock function or returns nil if not set
func (m *MockStore) GetBrewByRKey(ctx context.Context, rkey string) (*models.Brew, error) {
	if m.GetBrewByRKeyFunc != nil {
		return m.GetBrewByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

// ListBrews calls the mock function or returns empty slice if not set
func (m *MockStore) ListBrews(ctx context.Context, userID int) ([]*models.Brew, error) {
	if m.ListBrewsFunc != nil {
		return m.ListBrewsFunc(ctx, userID)
	}
	return []*models.Brew{}, nil
}

// UpdateBrewByRKey calls the mock function or returns nil if not set
func (m *MockStore) UpdateBrewByRKey(ctx context.Context, rkey string, brew *models.CreateBrewRequest) error {
	if m.UpdateBrewByRKeyFunc != nil {
		return m.UpdateBrewByRKeyFunc(ctx, rkey, brew)
	}
	return nil
}

// DeleteBrewByRKey calls the mock function or returns nil if not set
func (m *MockStore) DeleteBrewByRKey(ctx context.Context, rkey string) error {
	if m.DeleteBrewByRKeyFunc != nil {
		return m.DeleteBrewByRKeyFunc(ctx, rkey)
	}
	return nil
}

// CreateBean calls the mock function or returns nil if not set
func (m *MockStore) CreateBean(ctx context.Context, bean *models.CreateBeanRequest) (*models.Bean, error) {
	if m.CreateBeanFunc != nil {
		return m.CreateBeanFunc(ctx, bean)
	}
	return nil, nil
}

// GetBeanByRKey calls the mock function or returns nil if not set
func (m *MockStore) GetBeanByRKey(ctx context.Context, rkey string) (*models.Bean, error) {
	if m.GetBeanByRKeyFunc != nil {
		return m.GetBeanByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

// ListBeans calls the mock function or returns empty slice if not set
func (m *MockStore) ListBeans(ctx context.Context) ([]*models.Bean, error) {
	if m.ListBeansFunc != nil {
		return m.ListBeansFunc(ctx)
	}
	return []*models.Bean{}, nil
}

// UpdateBeanByRKey calls the mock function or returns nil if not set
func (m *MockStore) UpdateBeanByRKey(ctx context.Context, rkey string, bean *models.UpdateBeanRequest) error {
	if m.UpdateBeanByRKeyFunc != nil {
		return m.UpdateBeanByRKeyFunc(ctx, rkey, bean)
	}
	return nil
}

// DeleteBeanByRKey calls the mock function or returns nil if not set
func (m *MockStore) DeleteBeanByRKey(ctx context.Context, rkey string) error {
	if m.DeleteBeanByRKeyFunc != nil {
		return m.DeleteBeanByRKeyFunc(ctx, rkey)
	}
	return nil
}

// CreateRoaster calls the mock function or returns nil if not set
func (m *MockStore) CreateRoaster(ctx context.Context, roaster *models.CreateRoasterRequest) (*models.Roaster, error) {
	if m.CreateRoasterFunc != nil {
		return m.CreateRoasterFunc(ctx, roaster)
	}
	return nil, nil
}

// GetRoasterByRKey calls the mock function or returns nil if not set
func (m *MockStore) GetRoasterByRKey(ctx context.Context, rkey string) (*models.Roaster, error) {
	if m.GetRoasterByRKeyFunc != nil {
		return m.GetRoasterByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

// ListRoasters calls the mock function or returns empty slice if not set
func (m *MockStore) ListRoasters(ctx context.Context) ([]*models.Roaster, error) {
	if m.ListRoastersFunc != nil {
		return m.ListRoastersFunc(ctx)
	}
	return []*models.Roaster{}, nil
}

// UpdateRoasterByRKey calls the mock function or returns nil if not set
func (m *MockStore) UpdateRoasterByRKey(ctx context.Context, rkey string, roaster *models.UpdateRoasterRequest) error {
	if m.UpdateRoasterByRKeyFunc != nil {
		return m.UpdateRoasterByRKeyFunc(ctx, rkey, roaster)
	}
	return nil
}

// DeleteRoasterByRKey calls the mock function or returns nil if not set
func (m *MockStore) DeleteRoasterByRKey(ctx context.Context, rkey string) error {
	if m.DeleteRoasterByRKeyFunc != nil {
		return m.DeleteRoasterByRKeyFunc(ctx, rkey)
	}
	return nil
}

// CreateGrinder calls the mock function or returns nil if not set
func (m *MockStore) CreateGrinder(ctx context.Context, grinder *models.CreateGrinderRequest) (*models.Grinder, error) {
	if m.CreateGrinderFunc != nil {
		return m.CreateGrinderFunc(ctx, grinder)
	}
	return nil, nil
}

// GetGrinderByRKey calls the mock function or returns nil if not set
func (m *MockStore) GetGrinderByRKey(ctx context.Context, rkey string) (*models.Grinder, error) {
	if m.GetGrinderByRKeyFunc != nil {
		return m.GetGrinderByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

// ListGrinders calls the mock function or returns empty slice if not set
func (m *MockStore) ListGrinders(ctx context.Context) ([]*models.Grinder, error) {
	if m.ListGrindersFunc != nil {
		return m.ListGrindersFunc(ctx)
	}
	return []*models.Grinder{}, nil
}

// UpdateGrinderByRKey calls the mock function or returns nil if not set
func (m *MockStore) UpdateGrinderByRKey(ctx context.Context, rkey string, grinder *models.UpdateGrinderRequest) error {
	if m.UpdateGrinderByRKeyFunc != nil {
		return m.UpdateGrinderByRKeyFunc(ctx, rkey, grinder)
	}
	return nil
}

// DeleteGrinderByRKey calls the mock function or returns nil if not set
func (m *MockStore) DeleteGrinderByRKey(ctx context.Context, rkey string) error {
	if m.DeleteGrinderByRKeyFunc != nil {
		return m.DeleteGrinderByRKeyFunc(ctx, rkey)
	}
	return nil
}

// CreateBrewer calls the mock function or returns nil if not set
func (m *MockStore) CreateBrewer(ctx context.Context, brewer *models.CreateBrewerRequest) (*models.Brewer, error) {
	if m.CreateBrewerFunc != nil {
		return m.CreateBrewerFunc(ctx, brewer)
	}
	return nil, nil
}

// GetBrewerByRKey calls the mock function or returns nil if not set
func (m *MockStore) GetBrewerByRKey(ctx context.Context, rkey string) (*models.Brewer, error) {
	if m.GetBrewerByRKeyFunc != nil {
		return m.GetBrewerByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

// ListBrewers calls the mock function or returns empty slice if not set
func (m *MockStore) ListBrewers(ctx context.Context) ([]*models.Brewer, error) {
	if m.ListBrewersFunc != nil {
		return m.ListBrewersFunc(ctx)
	}
	return []*models.Brewer{}, nil
}

// UpdateBrewerByRKey calls the mock function or returns nil if not set
func (m *MockStore) UpdateBrewerByRKey(ctx context.Context, rkey string, brewer *models.UpdateBrewerRequest) error {
	if m.UpdateBrewerByRKeyFunc != nil {
		return m.UpdateBrewerByRKeyFunc(ctx, rkey, brewer)
	}
	return nil
}

// DeleteBrewerByRKey calls the mock function or returns nil if not set
func (m *MockStore) DeleteBrewerByRKey(ctx context.Context, rkey string) error {
	if m.DeleteBrewerByRKeyFunc != nil {
		return m.DeleteBrewerByRKeyFunc(ctx, rkey)
	}
	return nil
}

// Close calls the mock function or returns nil if not set
func (m *MockStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
