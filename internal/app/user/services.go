package user

import (
	"context"
	"fmt"

	"github.com/gotasma/internal/app/auth"
	"github.com/gotasma/internal/app/status"
	"github.com/gotasma/internal/app/types"
	"github.com/gotasma/internal/pkg/db"
	"github.com/gotasma/internal/pkg/uuid"
	"github.com/gotasma/internal/pkg/validator"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type (
	Repository interface {
		Create(context.Context, *types.User) (string, error)
		FindByEmail(ctx context.Context, email string) (*types.User, error)
		FindAllDev(ctx context.Context, createrID string) ([]*types.User, error)
		FindDevsByID(ctx context.Context, userIDs []string) ([]*types.User, error)
		Delete(cxt context.Context, id string) error
		FindByID(ctx context.Context, UserID string) (*types.User, error)
	}

	PolicyService interface {
		Validate(ctx context.Context, obj string, act string) error
	}

	ProjectService interface {
		RemoveDevs(ctx context.Context, userID string) error
	}

	Service struct {
		repo    Repository
		policy  PolicyService
		project ProjectService
	}
)

func New(repo Repository, policy PolicyService, project ProjectService) *Service {
	return &Service{
		repo:    repo,
		policy:  policy,
		project: project,
	}
}

func (s *Service) Register(ctx context.Context, req *types.RegisterRequest) (*types.User, error) {

	if err := validator.Validate(req); err != nil {
		logrus.Errorf("Fail to register PM due to invalid req, %w", err)
		validateErr := err.Error()
		return nil, fmt.Errorf(validateErr+"err: %w", status.Gen().BadRequest)
	}

	existingUser, err := s.repo.FindByEmail(ctx, req.Email)

	if err != nil && !db.IsErrNotFound(err) {
		logrus.Errorf("failed to check existing PM by email, err: %v", err)
		return nil, fmt.Errorf("failed to check existing user by email: %w", err)
	}

	if existingUser != nil {
		logrus.Error("PM email already registered")
		logrus.WithContext(ctx).Debug("email already registered")
		return nil, status.User().DuplicatedEmail
	}

	password, err := s.GeneratePassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	userID := uuid.New()

	user := &types.User{
		Password:  password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Role:      types.PM,
		UserID:    userID,
		CreaterID: userID,
	}

	if _, err := s.repo.Create(ctx, user); err != nil {
		logrus.Errorf("failed to insert PM, %v", err)
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return user.Strip(), nil
}

func (s *Service) CreateDev(ctx context.Context, req *types.RegisterRequest) (*types.User, error) {

	//Only PM can create dev
	if err := s.policy.Validate(ctx, types.PolicyObjectAny, types.PolicyActionAny); err != nil {
		return nil, err
	}
	if err := validator.Validate(req); err != nil {
		logrus.Errorf("Fail to register DEV due to invalid req, %v", err)
		validateErr := err.Error()
		return nil, fmt.Errorf(validateErr+"err: %w", status.Gen().BadRequest)
	}

	existingUser, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil && !db.IsErrNotFound(err) {
		logrus.Errorf("failed to check existing DEV by email, err: %v", err)
		return nil, fmt.Errorf("failed to check existing user by email: %w", err)
	}

	if existingUser != nil {
		logrus.Warning("Devs email is unique in database, different PM cannot have same DEV email account")
		logrus.Infof("Devs email already registered in system")
		return nil, status.User().DuplicatedEmail
	}

	password, err := s.GeneratePassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	pm := auth.FromContext(ctx)

	user := &types.User{
		Password:  password,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     req.Email,
		Role:      req.Role,
		UserID:    uuid.New(),
		CreaterID: pm.UserID,
	}

	if _, err := s.repo.Create(ctx, user); err != nil {
		logrus.Errorf("failed to insert DEV, %v", err)
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return user.Strip(), nil
}

func (s *Service) Auth(ctx context.Context, email, password string) (*types.User, error) {
	user, err := s.repo.FindByEmail(ctx, email)
	if err != nil && !db.IsErrNotFound(err) {
		logrus.Errorf("failed to check existing user by email, err: %v", err)
		return nil, status.Gen().Internal
	}
	if db.IsErrNotFound(err) {
		logrus.Debugf("user not found, email: %s", email)
		return nil, status.User().NotFoundUser
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		logrus.Error("invalid password")
		return nil, status.Auth().InvalidUserPassword
	}
	return user.Strip(), nil
}

func (s *Service) FindAllDev(ctx context.Context) ([]*types.UserInfo, error) {

	if err := s.policy.Validate(ctx, types.PolicyObjectAny, types.PolicyActionAny); err != nil {
		return nil, err
	}

	pm := auth.FromContext(ctx)

	users, err := s.repo.FindAllDev(ctx, pm.UserID)
	if err != nil {
		logrus.Errorf("can not find devs of PM, err: %v", err)
		return nil, status.User().NotFoundUser
	}

	info := make([]*types.UserInfo, 0)
	for _, usr := range users {
		info = append(info, &types.UserInfo{
			Email:     usr.Email,
			FirstName: usr.FirstName,
			LastName:  usr.LastName,
			Role:      usr.Role,
			CreaterID: usr.CreaterID,
			UserID:    usr.UserID,
			CreatedAt: usr.CreatedAt,
			UpdateAt:  usr.UpdateAt,
		})
	}
	return info, nil
}

//TODO: remove Devs_id in project
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.policy.Validate(ctx, types.PolicyObjectAny, types.PolicyActionAny); err != nil {
		return err
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil && !db.IsErrNotFound(err) {
		logrus.Errorf("failed to check existing user by ID, err: %v", err)
		return err
	}

	if db.IsErrNotFound(err) {
		logrus.Errorf("User doesn't exist, err: %v", err)
		return status.User().NotFoundUser
	}
	if user.Role == types.PM {
		logrus.Warning("This is PM_ID, cannot delete PM account, how can you get a pm_ID?")
		return status.Sercurity().InvalidAction
	}

	if err := s.project.RemoveDevs(ctx, id); err != nil {
		logrus.Errorf("Fail to remove Dev from project due to %v", err)
		return fmt.Errorf("Cannot remove Dev from projects, err:%w", err)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) CheckUsersExist(ctx context.Context, userID string) (string, error) {

	_, err := s.repo.FindByID(ctx, userID)

	if err != nil && !db.IsErrNotFound(err) {
		logrus.Errorf("failed to check existing user by ID, err: %v", err)
		return userID, err
	}

	if db.IsErrNotFound(err) {
		logrus.Errorf("User doesn't exist, err: %v", err)
		return userID, status.User().NotFoundUser
	}

	return "", nil
}

func (s *Service) GetDevsInfo(ctx context.Context, userIDs []string) ([]*types.UserInfo, error) {

	//TODO update project info DevsID if not found
	users, err := s.repo.FindDevsByID(ctx, userIDs)

	if err != nil && !db.IsErrNotFound(err) {
		logrus.Errorf("failed to check existing user by ID, err: %v", err)
		return nil, err
	}

	if db.IsErrNotFound(err) {
		logrus.Errorf("User doesn't exist, err: %v", err)
		return nil, status.User().NotFoundUser
	}

	info := make([]*types.UserInfo, 0)
	for _, usr := range users {
		info = append(info, &types.UserInfo{
			Email:     usr.Email,
			FirstName: usr.FirstName,
			LastName:  usr.LastName,
			Role:      usr.Role,
			CreaterID: usr.CreaterID,
			UserID:    usr.UserID,
			CreatedAt: usr.CreatedAt,
			UpdateAt:  usr.UpdateAt,
		})
	}
	return info, nil
}

func (s *Service) GeneratePassword(pass string) (string, error) {
	rs, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("failed to check hash password, %v", err)
		return "", fmt.Errorf("failed to generate password: %w", err)
	}
	return string(rs), nil
}
