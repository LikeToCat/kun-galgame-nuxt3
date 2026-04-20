package service

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	userModel "kun-galgame-api/internal/user/model"
)

// decodeProviderNames safely turns the row's jsonb provider_name bytes into a
// string slice. A null/empty/invalid value yields an empty slice rather than
// nil so the JSON response stays `[]` (frontend-friendly).
func decodeProviderNames(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal(raw, &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

// collectIDs extracts galgame IDs and user IDs from a list of resource rows.
func collectIDs(rows []model.GalgameResourceRow) (galgameIDs, userIDs []int) {
	galgameIDs = make([]int, 0, len(rows))
	userIDs = make([]int, 0, len(rows))
	for _, r := range rows {
		galgameIDs = append(galgameIDs, r.GalgameID)
		userIDs = append(userIDs, r.UserID)
	}
	return
}

// collectAggregate unions DISTINCT platform/language/type tuples into slices.
func collectAggregate(aggs []model.ResourceAggregate) (platforms, languages, types []string) {
	platforms, languages, types = []string{}, []string{}, []string{}
	for _, a := range aggs {
		if a.Platform != "" {
			platforms = appendUniqueStr(platforms, a.Platform)
		}
		if a.Language != "" {
			languages = appendUniqueStr(languages, a.Language)
		}
		if a.Type != "" {
			types = appendUniqueStr(types, a.Type)
		}
	}
	return
}

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

// briefToName maps a wiki GalgameBrief to the four-language KunLanguage DTO.
func briefToName(b client.GalgameBrief) dto.KunLanguage {
	return dto.KunLanguage{
		EnUs: b.NameEnUs, JaJp: b.NameJaJp,
		ZhCn: b.NameZhCn, ZhTw: b.NameZhTw,
	}
}

// userBriefToDTO maps a user model to the dto.UserBrief projection.
func userBriefToDTO(u userModel.UserBrief) dto.UserBrief {
	return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
}

// rowToCard maps a resource row to the list-card DTO.
func rowToCard(r model.GalgameResourceRow, u userModel.UserBrief) dto.ResourceCard {
	return dto.ResourceCard{
		ID:            r.ID,
		View:          r.View,
		GalgameID:     r.GalgameID,
		User:          userBriefToDTO(u),
		Type:          r.Type,
		Language:      r.Language,
		Platform:      r.Platform,
		Size:          r.Size,
		Status:        r.Status,
		Download:      r.Download,
		LikeCount:     r.LikeCount,
		IsLiked:       false,
		LinkDomain:    "",
		ProviderNames: decodeProviderNames(r.ProviderName),
		Note:          r.Note,
		Created:       r.Created,
		Edited:        r.Edited,
	}
}

// rowToDownloadDetail maps a resource row + links + liked flag + owner to the
// download-detail DTO.
func rowToDownloadDetail(
	r model.GalgameResourceRow,
	links []string,
	isLiked bool,
	owner userModel.UserBrief,
) dto.ResourceDownloadDetail {
	linkDomain := ""
	if len(links) > 0 {
		linkDomain = links[0]
	}
	return dto.ResourceDownloadDetail{
		ID:            r.ID,
		View:          r.View,
		GalgameID:     r.GalgameID,
		User:          userBriefToDTO(owner),
		Type:          r.Type,
		Language:      r.Language,
		Platform:      r.Platform,
		Size:          r.Size,
		Status:        r.Status,
		Download:      r.Download,
		LikeCount:     r.LikeCount,
		IsLiked:       isLiked,
		LinkDomain:    linkDomain,
		ProviderNames: decodeProviderNames(r.ProviderName),
		Link:          links,
		Code:          r.Code,
		Password:      r.Password,
		Note:          r.Note,
		Created:       r.Created,
		Edited:        r.Edited,
	}
}
