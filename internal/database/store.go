package database

import (
	"context"

	"arabica/internal/models"
)

// Store defines the interface for all database operations.
// This abstraction allows swapping SQLite for ATProto or other backends.
// All methods accept a context.Context as the first parameter to support
// cancellation, timeouts, and request-scoped values.
type Store interface {
	// Brew operations
	// Note: userID parameter is deprecated for ATProto (user is implicit from DID)
	// It remains for SQLite compatibility but should not be relied upon
	CreateBrew(ctx context.Context, brew *models.CreateBrewRequest, userID int) (*models.Brew, error)
	GetBrewByRKey(ctx context.Context, rkey string) (*models.Brew, error)
	ListBrews(ctx context.Context, userID int) ([]*models.Brew, error)
	UpdateBrewByRKey(ctx context.Context, rkey string, brew *models.CreateBrewRequest) error
	DeleteBrewByRKey(ctx context.Context, rkey string) error

	// Bean operations
	CreateBean(ctx context.Context, bean *models.CreateBeanRequest) (*models.Bean, error)
	GetBeanByRKey(ctx context.Context, rkey string) (*models.Bean, error)
	ListBeans(ctx context.Context) ([]*models.Bean, error)
	UpdateBeanByRKey(ctx context.Context, rkey string, bean *models.UpdateBeanRequest) error
	DeleteBeanByRKey(ctx context.Context, rkey string) error

	// Roaster operations
	CreateRoaster(ctx context.Context, roaster *models.CreateRoasterRequest) (*models.Roaster, error)
	GetRoasterByRKey(ctx context.Context, rkey string) (*models.Roaster, error)
	ListRoasters(ctx context.Context) ([]*models.Roaster, error)
	UpdateRoasterByRKey(ctx context.Context, rkey string, roaster *models.UpdateRoasterRequest) error
	DeleteRoasterByRKey(ctx context.Context, rkey string) error

	// Grinder operations
	CreateGrinder(ctx context.Context, grinder *models.CreateGrinderRequest) (*models.Grinder, error)
	GetGrinderByRKey(ctx context.Context, rkey string) (*models.Grinder, error)
	ListGrinders(ctx context.Context) ([]*models.Grinder, error)
	UpdateGrinderByRKey(ctx context.Context, rkey string, grinder *models.UpdateGrinderRequest) error
	DeleteGrinderByRKey(ctx context.Context, rkey string) error

	// Brewer operations
	CreateBrewer(ctx context.Context, brewer *models.CreateBrewerRequest) (*models.Brewer, error)
	GetBrewerByRKey(ctx context.Context, rkey string) (*models.Brewer, error)
	ListBrewers(ctx context.Context) ([]*models.Brewer, error)
	UpdateBrewerByRKey(ctx context.Context, rkey string, brewer *models.UpdateBrewerRequest) error
	DeleteBrewerByRKey(ctx context.Context, rkey string) error

	// Close the database connection
	Close() error
}
