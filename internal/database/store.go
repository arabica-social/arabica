package database

import (
	"context"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
)

// Store defines the interface for all database operations.
// This abstraction allows swapping SQLite for ATProto or other backends.
// All methods accept a context.Context as the first parameter to support
// cancellation, timeouts, and request-scoped values.
type Store interface {
	// Brew operations
	// Note: userID parameter is deprecated for ATProto (user is implicit from DID)
	// It remains for SQLite compatibility but should not be relied upon
	CreateBrew(ctx context.Context, brew *arabica.CreateBrewRequest, userID int) (*arabica.Brew, error)
	GetBrewByRKey(ctx context.Context, rkey string) (*arabica.Brew, error)
	ListBrews(ctx context.Context, userID int) ([]*arabica.Brew, error)
	UpdateBrewByRKey(ctx context.Context, rkey string, brew *arabica.CreateBrewRequest) error
	DeleteBrewByRKey(ctx context.Context, rkey string) error

	// Bean operations
	CreateBean(ctx context.Context, bean *arabica.CreateBeanRequest) (*arabica.Bean, error)
	GetBeanByRKey(ctx context.Context, rkey string) (*arabica.Bean, error)
	ListBeans(ctx context.Context) ([]*arabica.Bean, error)
	UpdateBeanByRKey(ctx context.Context, rkey string, bean *arabica.UpdateBeanRequest) error
	DeleteBeanByRKey(ctx context.Context, rkey string) error

	// Roaster operations
	CreateRoaster(ctx context.Context, roaster *arabica.CreateRoasterRequest) (*arabica.Roaster, error)
	GetRoasterByRKey(ctx context.Context, rkey string) (*arabica.Roaster, error)
	ListRoasters(ctx context.Context) ([]*arabica.Roaster, error)
	UpdateRoasterByRKey(ctx context.Context, rkey string, roaster *arabica.UpdateRoasterRequest) error
	DeleteRoasterByRKey(ctx context.Context, rkey string) error

	// Grinder operations
	CreateGrinder(ctx context.Context, grinder *arabica.CreateGrinderRequest) (*arabica.Grinder, error)
	GetGrinderByRKey(ctx context.Context, rkey string) (*arabica.Grinder, error)
	ListGrinders(ctx context.Context) ([]*arabica.Grinder, error)
	UpdateGrinderByRKey(ctx context.Context, rkey string, grinder *arabica.UpdateGrinderRequest) error
	DeleteGrinderByRKey(ctx context.Context, rkey string) error

	// Brewer operations
	CreateBrewer(ctx context.Context, brewer *arabica.CreateBrewerRequest) (*arabica.Brewer, error)
	GetBrewerByRKey(ctx context.Context, rkey string) (*arabica.Brewer, error)
	ListBrewers(ctx context.Context) ([]*arabica.Brewer, error)
	UpdateBrewerByRKey(ctx context.Context, rkey string, brewer *arabica.UpdateBrewerRequest) error
	DeleteBrewerByRKey(ctx context.Context, rkey string) error

	// Recipe operations
	CreateRecipe(ctx context.Context, recipe *arabica.CreateRecipeRequest) (*arabica.Recipe, error)
	GetRecipeByRKey(ctx context.Context, rkey string) (*arabica.Recipe, error)
	ListRecipes(ctx context.Context) ([]*arabica.Recipe, error)
	UpdateRecipeByRKey(ctx context.Context, rkey string, recipe *arabica.UpdateRecipeRequest) error
	DeleteRecipeByRKey(ctx context.Context, rkey string) error

	// Like operations
	CreateLike(ctx context.Context, req *arabica.CreateLikeRequest) (*arabica.Like, error)
	DeleteLikeByRKey(ctx context.Context, rkey string) error
	GetUserLikeForSubject(ctx context.Context, subjectURI string) (*arabica.Like, error)
	ListUserLikes(ctx context.Context) ([]*arabica.Like, error)

	// Comment operations
	CreateComment(ctx context.Context, req *arabica.CreateCommentRequest) (*arabica.Comment, error)
	DeleteCommentByRKey(ctx context.Context, rkey string) error
	GetCommentsForSubject(ctx context.Context, subjectURI string) ([]*arabica.Comment, error)
	ListUserComments(ctx context.Context) ([]*arabica.Comment, error)

	// Close the database connection
	Close() error
}
