package holiday

import (
	"context"
	"fmt"

	"github.com/gotasma/internal/app/auth"
	"github.com/gotasma/internal/app/status"
	"github.com/gotasma/internal/app/types"
	"github.com/gotasma/internal/pkg/db"
	"github.com/gotasma/internal/pkg/uuid"
	"github.com/gotasma/internal/pkg/validator"
)

const (
	MilisecondInDay = 86400000
)

type (
	Repository interface {
		Create(ctx context.Context, holiday *types.Holiday) error
		FindByTitle(ctx context.Context, title string, createrID string) (*types.Holiday, error)
		Delete(ctx context.Context, id string) error
		FindAll(ctx context.Context, createrID string) ([]*types.Holiday, error)
	}
	PolicyServices interface {
		Validate(ctx context.Context, obj string, act string) error
	}
	Services struct {
		repo   Repository
		policy PolicyServices
	}
)

func New(repo Repository, policy PolicyServices) *Services {
	return &Services{
		repo:   repo,
		policy: policy,
	}
}

func (s *Services) Create(ctx context.Context, req *types.HolidayRequest) (*types.Holiday, error) {
	pm := auth.FromContext(ctx)

	if err := s.policy.Validate(ctx, types.ObjectHoliday, types.ActionHoliday); err != nil {
		return nil, err
	}

	if err := validator.Validate(req); err != nil {
		return nil, err
	}

	existingHoliday, err := s.repo.FindByTitle(ctx, req.Title, pm.UserID)
	if err != nil && !db.IsErrNotFound(err) {
		return nil, fmt.Errorf("failed to check existing holiday by title: %w", err)
	}
	if existingHoliday != nil {
		return nil, status.Hoiday().DuplicatedHoliday
	}

	holiday := &types.Holiday{
		Title:     req.Title,
		Start:     req.Start,
		End:       req.End,
		HolidayID: uuid.New(),
		Duration:  ((req.End - req.Start) / MilisecondInDay),
		CreaterID: pm.UserID,
	}
	if err := s.repo.Create(ctx, holiday); err != nil {
		return nil, fmt.Errorf("Faild to insert Holiday, %w", err)
	}
	return holiday, nil
}

func (s *Services) Delete(ctx context.Context, id string) error {
	if err := s.policy.Validate(ctx, types.PolicyObjectDeleteDev, types.PolicyActionDevDelete); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, id); err != nil {
		return status.Gen().NotFound
	}
	return nil
}

func (s *Services) FindAll(ctx context.Context) ([]*types.Holiday, error) {

	user := auth.FromContext(ctx)

	var holidays []*types.Holiday
	var err error

	//Check current client roles, pass different id to func depends on role
	userID := user.UserID
	if user.Role != types.PM {
		userID = user.CreaterID
	}

	holidays, err = s.repo.FindAll(ctx, userID)

	info := make([]*types.Holiday, 0)
	for _, holiday := range holidays {
		info = append(info, &types.Holiday{
			Title:     holiday.Title,
			Start:     holiday.Start,
			End:       holiday.End,
			Duration:  holiday.Duration,
			HolidayID: holiday.HolidayID,
		})
	}
	return info, err
}
