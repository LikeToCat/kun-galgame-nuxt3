package service

import (
	"context"

	"kun-galgame-api/internal/admin/dto"
	"kun-galgame-api/internal/admin/repository"
)

type SettingService struct {
	settingRepo *repository.SettingRepository
}

func NewSettingService(settingRepo *repository.SettingRepository) *SettingService {
	return &SettingService{settingRepo: settingRepo}
}

// GetRegisterSetting — GET /admin/setting/register
// `registerStatus` is true when registration is OPEN.
func (s *SettingService) GetRegisterSetting(ctx context.Context) *dto.RegisterSettingResponse {
	disabled := s.settingRepo.GetRegisterDisabled(ctx)
	return &dto.RegisterSettingResponse{RegisterStatus: !disabled}
}

// ToggleRegisterSetting — PUT /admin/setting/register
func (s *SettingService) ToggleRegisterSetting(ctx context.Context) {
	s.settingRepo.ToggleRegisterDisabled(ctx)
}
