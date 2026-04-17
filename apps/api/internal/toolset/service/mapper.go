package service

import (
	"math"

	"kun-galgame-api/internal/toolset/dto"
	"kun-galgame-api/internal/toolset/model"
	userModel "kun-galgame-api/internal/user/model"
)

// toolsetCardFromRow maps a raw toolset row to a list-card DTO given the
// pre-loaded batch maps.
func toolsetCardFromRow(
	t model.GalgameToolset,
	userMap map[int]userModel.UserBrief,
	avgMap map[int]float64,
	dlMap map[int]int,
	ccMap map[int]int,
) dto.ToolsetCard {
	var practicalityAvg any
	if avg, ok := avgMap[t.ID]; ok && avg != 0 {
		practicalityAvg = math.Round(avg*100) / 100
	} else {
		practicalityAvg = nil
	}

	return dto.ToolsetCard{
		ID:                 t.ID,
		Name:               t.Name,
		User:               userMap[t.UserID],
		Type:               t.Type,
		Platform:           t.Platform,
		Language:           t.Language,
		Version:            t.Version,
		View:               t.View,
		Download:           dlMap[t.ID],
		CommentCount:       ccMap[t.ID],
		PracticalityAvg:    practicalityAvg,
		ResourceUpdateTime: t.ResourceUpdateTime,
	}
}

// allowedSortField returns the DB column name to sort by and whether the value
// is allowed. Defaults to "created" when the input is empty or not in the
// whitelist.
func allowedSortField(raw string) string {
	allowed := map[string]bool{
		"created":              true,
		"view":                 true,
		"name":                 true,
		"resource_update_time": true,
	}
	if raw != "" && allowed[raw] {
		return raw
	}
	return "created"
}

// sortOrder normalises the sort order (only "asc" flips to ASC).
func sortOrder(raw string) string {
	if raw == "asc" {
		return "ASC"
	}
	return "DESC"
}
