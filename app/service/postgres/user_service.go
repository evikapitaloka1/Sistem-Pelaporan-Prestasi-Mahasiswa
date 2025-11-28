package service

import (
	"context"
	"errors"

	"uas/app/model/postgres"
	"uas/app/repository/postgres"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}


func (s *UserService) GetAllUsers(ctx context.Context) ([]model.User, error) {
	return s.repo.ListUsers(ctx)
}


func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.repo.GetUserByID(ctx, id)
}


func (s *UserService) CreateUser(ctx context.Context, req model.CreateUserRequest) (uuid.UUID, error) {
	// hash password sebelum disimpan
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, err
	}

	req.Password = string(hashedPassword)

	return s.repo.CreateUser(ctx, req)
}

func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) error {
	return s.repo.UpdateUser(ctx, id, req)
}

func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}
func (s *UserService) UpdateUserRole(ctx context.Context, id uuid.UUID, roleID uuid.UUID) error {
	if roleID == uuid.Nil {
		return errors.New("role_id tidak boleh kosong")
	}

	return s.repo.UpdateUserRole(ctx, id, roleID)
}

