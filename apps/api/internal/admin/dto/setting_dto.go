package dto

// RegisterSettingResponse is the shape of GET /admin/setting/register.
type RegisterSettingResponse struct {
	RegisterStatus bool `json:"registerStatus"`
}
