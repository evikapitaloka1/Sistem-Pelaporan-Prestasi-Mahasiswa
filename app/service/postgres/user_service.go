package service

import (
	"context"
	"errors"

	"uas/app/model/postgres"
	"uas/app/repository/postgres"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// ================= INTERFACE =================
type IUserService interface {
	GetAllUsers(ctx context.Context) ([]model.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error)
	CreateUser(ctx context.Context, req model.CreateUserRequest) (uuid.UUID, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	UpdateUserRole(ctx context.Context, id uuid.UUID, roleID uuid.UUID) error
}

// ================= STRUCT =================
type UserService struct {
	repo repository.UserRepository
}

// pastikan implement interface
var _ IUserService = &UserService{}

// ================= CONSTRUCTOR =================
func NewUserService(repo repository.UserRepository) *UserService {
	return &UserService{repo: repo}
}

// ================= GET ALL USERS =================
func (s *UserService) GetAllUsers(ctx context.Context) ([]model.User, error) {
	return s.repo.ListUsers(ctx)
}

// ================= GET USER BY ID =================
func (s *UserService) GetUserByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

// ================= CREATE USER =================
func (s *UserService) CreateUser(ctx context.Context, req model.CreateUserRequest) (uuid.UUID, error) {
	// hash password sebelum disimpan
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return uuid.Nil, err
	}
	req.Password = string(hashedPassword)

	return s.repo.CreateUser(ctx, req)
}

// ================= UPDATE USER =================
func (s *UserService) UpdateUser(ctx context.Context, id uuid.UUID, req model.UpdateUserRequest) error {
	return s.repo.UpdateUser(ctx, id, req)
}

// ================= DELETE USER =================
func (s *UserService) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}

// ================= UPDATE USER ROLE =================
func (s *UserService) UpdateUserRole(ctx context.Context, id uuid.UUID, roleID uuid.UUID) error {
	if roleID == uuid.Nil {
		return errors.New("role_id tidak boleh kosong")
	}
	return s.repo.UpdateUserRole(ctx, id, roleID)
}
