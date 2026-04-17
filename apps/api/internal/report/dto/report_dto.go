package dto

type SubmitReportRequest struct {
	Reason string `json:"reason" validate:"required,max=1000"`
	Type   string `json:"type" validate:"required,max=100"`
}
