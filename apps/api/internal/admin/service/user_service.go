package service

import (
	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/repository"
)

type UserService struct {
	userRepo *repository.UserRepository
}

func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// GetUserList — GET /admin/user
func (s *UserService) GetUserList(page, limit int) *dto.UserListResponse {
	return &dto.UserListResponse{
		Users:      s.userRepo.FindPaginated(page, limit),
		TotalCount: s.userRepo.CountAll(),
	}
}

// SearchUsers — GET /admin/user/search
func (s *UserService) SearchUsers(keyword string) []dto.AdminUserRow {
	return s.userRepo.SearchByName(keyword)
}
