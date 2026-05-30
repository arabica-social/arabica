package arabicastore

import (
	"context"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/records"
)

// MockStore is a mock implementation of the Store interface for testing.
// Uses function fields to allow tests to inject custom behavior.
type MockStore struct {
	CreateBrewFunc       func(ctx context.Context, brew *arabica.CreateBrewRequest, userID int) (*arabica.Brew, error)
	GetBrewByRKeyFunc    func(ctx context.Context, rkey string) (*arabica.Brew, error)
	ListBrewsFunc        func(ctx context.Context, userID int, offset, limit int) ([]*arabica.Brew, error)
	UpdateBrewByRKeyFunc func(ctx context.Context, rkey string, brew *arabica.CreateBrewRequest) error
	DeleteBrewByRKeyFunc func(ctx context.Context, rkey string) error

	CreateBeanFunc       func(ctx context.Context, bean *arabica.CreateBeanRequest) (*arabica.Bean, error)
	GetBeanByRKeyFunc    func(ctx context.Context, rkey string) (*arabica.Bean, error)
	ListBeansFunc        func(ctx context.Context) ([]*arabica.Bean, error)
	UpdateBeanByRKeyFunc func(ctx context.Context, rkey string, bean *arabica.UpdateBeanRequest) error
	DeleteBeanByRKeyFunc func(ctx context.Context, rkey string) error

	CreateRoasterFunc       func(ctx context.Context, roaster *arabica.CreateRoasterRequest) (*arabica.Roaster, error)
	GetRoasterByRKeyFunc    func(ctx context.Context, rkey string) (*arabica.Roaster, error)
	ListRoastersFunc        func(ctx context.Context) ([]*arabica.Roaster, error)
	UpdateRoasterByRKeyFunc func(ctx context.Context, rkey string, roaster *arabica.UpdateRoasterRequest) error
	DeleteRoasterByRKeyFunc func(ctx context.Context, rkey string) error

	ListGrindersFunc func(ctx context.Context) ([]*arabica.Grinder, error)

	ListBrewersFunc func(ctx context.Context) ([]*arabica.Brewer, error)

	CreateRecipeFunc       func(ctx context.Context, recipe *arabica.CreateRecipeRequest) (*arabica.Recipe, error)
	GetRecipeByRKeyFunc    func(ctx context.Context, rkey string) (*arabica.Recipe, error)
	ListRecipesFunc        func(ctx context.Context) ([]*arabica.Recipe, error)
	UpdateRecipeByRKeyFunc func(ctx context.Context, rkey string, recipe *arabica.UpdateRecipeRequest) error
	DeleteRecipeByRKeyFunc func(ctx context.Context, rkey string) error

	CreateLikeFunc            func(ctx context.Context, req *arabica.CreateLikeRequest) (*arabica.Like, error)
	DeleteLikeByRKeyFunc      func(ctx context.Context, rkey string) error
	GetUserLikeForSubjectFunc func(ctx context.Context, subjectURI string) (*arabica.Like, error)
	ListUserLikesFunc         func(ctx context.Context) ([]*arabica.Like, error)

	CreateCommentFunc         func(ctx context.Context, req *arabica.CreateCommentRequest) (*arabica.Comment, error)
	DeleteCommentByRKeyFunc   func(ctx context.Context, rkey string) error
	GetCommentsForSubjectFunc func(ctx context.Context, subjectURI string) ([]*arabica.Comment, error)
	ListUserCommentsFunc      func(ctx context.Context) ([]*arabica.Comment, error)

	DIDFunc             func() string
	FetchRecordFunc     func(ctx context.Context, nsid, rkey string) (record map[string]any, uri, cid string, err error)
	FetchAllRecordsFunc func(ctx context.Context, nsid string) ([]records.RawRecord, error)
	PutRecordFunc       func(ctx context.Context, nsid, rkey string, record any) (resultRKey, cid string, err error)
	RemoveRecordFunc    func(ctx context.Context, nsid, rkey string) error

	CloseFunc func() error
}

func (m *MockStore) CreateBrew(ctx context.Context, brew *arabica.CreateBrewRequest, userID int) (*arabica.Brew, error) {
	if m.CreateBrewFunc != nil {
		return m.CreateBrewFunc(ctx, brew, userID)
	}
	return nil, nil
}

func (m *MockStore) GetBrewByRKey(ctx context.Context, rkey string) (*arabica.Brew, error) {
	if m.GetBrewByRKeyFunc != nil {
		return m.GetBrewByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

func (m *MockStore) ListBrews(ctx context.Context, userID int, offset, limit int) ([]*arabica.Brew, error) {
	if m.ListBrewsFunc != nil {
		return m.ListBrewsFunc(ctx, userID, offset, limit)
	}
	return []*arabica.Brew{}, nil
}

func (m *MockStore) UpdateBrewByRKey(ctx context.Context, rkey string, brew *arabica.CreateBrewRequest) error {
	if m.UpdateBrewByRKeyFunc != nil {
		return m.UpdateBrewByRKeyFunc(ctx, rkey, brew)
	}
	return nil
}

func (m *MockStore) DeleteBrewByRKey(ctx context.Context, rkey string) error {
	if m.DeleteBrewByRKeyFunc != nil {
		return m.DeleteBrewByRKeyFunc(ctx, rkey)
	}
	return nil
}

func (m *MockStore) CreateBean(ctx context.Context, bean *arabica.CreateBeanRequest) (*arabica.Bean, error) {
	if m.CreateBeanFunc != nil {
		return m.CreateBeanFunc(ctx, bean)
	}
	return nil, nil
}

func (m *MockStore) GetBeanByRKey(ctx context.Context, rkey string) (*arabica.Bean, error) {
	if m.GetBeanByRKeyFunc != nil {
		return m.GetBeanByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

func (m *MockStore) ListBeans(ctx context.Context) ([]*arabica.Bean, error) {
	if m.ListBeansFunc != nil {
		return m.ListBeansFunc(ctx)
	}
	return []*arabica.Bean{}, nil
}

func (m *MockStore) UpdateBeanByRKey(ctx context.Context, rkey string, bean *arabica.UpdateBeanRequest) error {
	if m.UpdateBeanByRKeyFunc != nil {
		return m.UpdateBeanByRKeyFunc(ctx, rkey, bean)
	}
	return nil
}

func (m *MockStore) DeleteBeanByRKey(ctx context.Context, rkey string) error {
	if m.DeleteBeanByRKeyFunc != nil {
		return m.DeleteBeanByRKeyFunc(ctx, rkey)
	}
	return nil
}

func (m *MockStore) CreateRoaster(ctx context.Context, roaster *arabica.CreateRoasterRequest) (*arabica.Roaster, error) {
	if m.CreateRoasterFunc != nil {
		return m.CreateRoasterFunc(ctx, roaster)
	}
	return nil, nil
}

func (m *MockStore) GetRoasterByRKey(ctx context.Context, rkey string) (*arabica.Roaster, error) {
	if m.GetRoasterByRKeyFunc != nil {
		return m.GetRoasterByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

func (m *MockStore) ListRoasters(ctx context.Context) ([]*arabica.Roaster, error) {
	if m.ListRoastersFunc != nil {
		return m.ListRoastersFunc(ctx)
	}
	return []*arabica.Roaster{}, nil
}

func (m *MockStore) UpdateRoasterByRKey(ctx context.Context, rkey string, roaster *arabica.UpdateRoasterRequest) error {
	if m.UpdateRoasterByRKeyFunc != nil {
		return m.UpdateRoasterByRKeyFunc(ctx, rkey, roaster)
	}
	return nil
}

func (m *MockStore) DeleteRoasterByRKey(ctx context.Context, rkey string) error {
	if m.DeleteRoasterByRKeyFunc != nil {
		return m.DeleteRoasterByRKeyFunc(ctx, rkey)
	}
	return nil
}

func (m *MockStore) ListGrinders(ctx context.Context) ([]*arabica.Grinder, error) {
	if m.ListGrindersFunc != nil {
		return m.ListGrindersFunc(ctx)
	}
	return []*arabica.Grinder{}, nil
}

func (m *MockStore) ListBrewers(ctx context.Context) ([]*arabica.Brewer, error) {
	if m.ListBrewersFunc != nil {
		return m.ListBrewersFunc(ctx)
	}
	return []*arabica.Brewer{}, nil
}

func (m *MockStore) CreateRecipe(ctx context.Context, recipe *arabica.CreateRecipeRequest) (*arabica.Recipe, error) {
	if m.CreateRecipeFunc != nil {
		return m.CreateRecipeFunc(ctx, recipe)
	}
	return nil, nil
}

func (m *MockStore) GetRecipeByRKey(ctx context.Context, rkey string) (*arabica.Recipe, error) {
	if m.GetRecipeByRKeyFunc != nil {
		return m.GetRecipeByRKeyFunc(ctx, rkey)
	}
	return nil, nil
}

func (m *MockStore) ListRecipes(ctx context.Context) ([]*arabica.Recipe, error) {
	if m.ListRecipesFunc != nil {
		return m.ListRecipesFunc(ctx)
	}
	return []*arabica.Recipe{}, nil
}

func (m *MockStore) UpdateRecipeByRKey(ctx context.Context, rkey string, recipe *arabica.UpdateRecipeRequest) error {
	if m.UpdateRecipeByRKeyFunc != nil {
		return m.UpdateRecipeByRKeyFunc(ctx, rkey, recipe)
	}
	return nil
}

func (m *MockStore) DeleteRecipeByRKey(ctx context.Context, rkey string) error {
	if m.DeleteRecipeByRKeyFunc != nil {
		return m.DeleteRecipeByRKeyFunc(ctx, rkey)
	}
	return nil
}

func (m *MockStore) CreateLike(ctx context.Context, req *arabica.CreateLikeRequest) (*arabica.Like, error) {
	if m.CreateLikeFunc != nil {
		return m.CreateLikeFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockStore) DeleteLikeByRKey(ctx context.Context, rkey string) error {
	if m.DeleteLikeByRKeyFunc != nil {
		return m.DeleteLikeByRKeyFunc(ctx, rkey)
	}
	return nil
}

func (m *MockStore) GetUserLikeForSubject(ctx context.Context, subjectURI string) (*arabica.Like, error) {
	if m.GetUserLikeForSubjectFunc != nil {
		return m.GetUserLikeForSubjectFunc(ctx, subjectURI)
	}
	return nil, nil
}

func (m *MockStore) ListUserLikes(ctx context.Context) ([]*arabica.Like, error) {
	if m.ListUserLikesFunc != nil {
		return m.ListUserLikesFunc(ctx)
	}
	return []*arabica.Like{}, nil
}

func (m *MockStore) CreateComment(ctx context.Context, req *arabica.CreateCommentRequest) (*arabica.Comment, error) {
	if m.CreateCommentFunc != nil {
		return m.CreateCommentFunc(ctx, req)
	}
	return nil, nil
}

func (m *MockStore) DeleteCommentByRKey(ctx context.Context, rkey string) error {
	if m.DeleteCommentByRKeyFunc != nil {
		return m.DeleteCommentByRKeyFunc(ctx, rkey)
	}
	return nil
}

func (m *MockStore) GetCommentsForSubject(ctx context.Context, subjectURI string) ([]*arabica.Comment, error) {
	if m.GetCommentsForSubjectFunc != nil {
		return m.GetCommentsForSubjectFunc(ctx, subjectURI)
	}
	return []*arabica.Comment{}, nil
}

func (m *MockStore) ListUserComments(ctx context.Context) ([]*arabica.Comment, error) {
	if m.ListUserCommentsFunc != nil {
		return m.ListUserCommentsFunc(ctx)
	}
	return []*arabica.Comment{}, nil
}

func (m *MockStore) DID() string {
	if m.DIDFunc != nil {
		return m.DIDFunc()
	}
	return "did:plc:abcdefghijklmnopqrstuvwx"
}

func (m *MockStore) FetchRecord(ctx context.Context, nsid, rkey string) (map[string]any, string, string, error) {
	if m.FetchRecordFunc != nil {
		return m.FetchRecordFunc(ctx, nsid, rkey)
	}
	return nil, "", "", nil
}

func (m *MockStore) FetchAllRecords(ctx context.Context, nsid string) ([]records.RawRecord, error) {
	if m.FetchAllRecordsFunc != nil {
		return m.FetchAllRecordsFunc(ctx, nsid)
	}
	switch nsid {
	case arabica.NSIDBrewer:
		if m.ListBrewersFunc == nil {
			return nil, nil
		}
		brewers, err := m.ListBrewersFunc(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]records.RawRecord, 0, len(brewers))
		for i, b := range brewers {
			rkey := b.RKey
			if rkey == "" {
				rkey = "brewer-test"
			}
			rec, err := arabica.BrewerToRecord(b)
			if err != nil {
				return nil, err
			}
			out = append(out, records.RawRecord{URI: "", RKey: rkey, CID: "test-cid", Record: rec})
			_ = i
		}
		return out, nil
	case arabica.NSIDGrinder:
		if m.ListGrindersFunc == nil {
			return nil, nil
		}
		grinders, err := m.ListGrindersFunc(ctx)
		if err != nil {
			return nil, err
		}
		out := make([]records.RawRecord, 0, len(grinders))
		for _, g := range grinders {
			rkey := g.RKey
			if rkey == "" {
				rkey = "grinder-test"
			}
			rec, err := arabica.GrinderToRecord(g)
			if err != nil {
				return nil, err
			}
			out = append(out, records.RawRecord{URI: "", RKey: rkey, CID: "test-cid", Record: rec})
		}
		return out, nil
	}
	return nil, nil
}

func (m *MockStore) PutRecord(ctx context.Context, nsid, rkey string, record any) (string, string, error) {
	if m.PutRecordFunc != nil {
		return m.PutRecordFunc(ctx, nsid, rkey, record)
	}
	if rkey != "" {
		return rkey, "", nil
	}
	return "test-rkey", "test-cid", nil
}

func (m *MockStore) RemoveRecord(ctx context.Context, nsid, rkey string) error {
	if m.RemoveRecordFunc != nil {
		return m.RemoveRecordFunc(ctx, nsid, rkey)
	}
	return nil
}

func (m *MockStore) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
