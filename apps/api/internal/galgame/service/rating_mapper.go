package service

import (
	"encoding/json"

	"kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/pkg/userclient"
)

// rawJSON wraps a DB-stored JSON string into a json.RawMessage, falling back
// to `null` for empty strings so the field serialises cleanly.
func rawJSON(s string) json.RawMessage {
	if s == "" {
		return json.RawMessage("null")
	}
	return json.RawMessage(s)
}

// rowToScores pulls the per-axis scores off a rating row.
func rowToScores(r model.GalgameRatingRow) dto.RatingScores {
	return dto.RatingScores{
		Art: r.Art, Story: r.Story, Music: r.Music, Character: r.Character,
		Route: r.Route, System: r.System, Voice: r.Voice,
		ReplayValue: r.ReplayValue,
	}
}

// ratingRowToCard maps a rating row + user + brief to the list card DTO.
func ratingRowToCard(
	r model.GalgameRatingRow,
	user userclient.User,
	brief client.GalgameBrief,
) dto.RatingCard {
	return dto.RatingCard{
		ID:           r.ID,
		User:         userBriefToDTO(user),
		Recommend:    r.Recommend,
		Overall:      r.Overall,
		View:         r.View,
		GalgameType:  rawJSON(r.GalgameType),
		PlayStatus:   r.PlayStatus,
		ShortSummary: r.ShortSummary,
		SpoilerLevel: r.SpoilerLevel,
		RatingScores: rowToScores(r),
		LikeCount:    r.LikeCount,
		Created:      r.Created,
		Updated:      r.Updated,
		Galgame: dto.RatingGalgameBrief{
			ID:           brief.ID,
			ContentLimit: brief.ContentLimit,
			Name: dto.KunLanguage{
				EnUs: brief.NameEnUs, JaJp: brief.NameJaJp,
				ZhCn: brief.NameZhCn, ZhTw: brief.NameZhTw,
			},
		},
	}
}

// ratingCommentRowToDTO maps a comment row + hydrated identity map to the
// response item. The userMap is keyed by user_id and produced via userclient.
func ratingCommentRowToDTO(cm model.RatingCommentRow, userMap map[int]userclient.User) dto.RatingCommentItem {
	u := userMap[cm.UserID]
	item := dto.RatingCommentItem{
		ID:      cm.ID,
		Content: cm.Content,
		User:    dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar},
		Created: cm.Created,
		Updated: cm.Updated,
	}
	if cm.TargetUserID != nil {
		t := userMap[*cm.TargetUserID]
		item.TargetUser = &dto.UserBrief{
			ID: t.ID, Name: t.Name, Avatar: t.Avatar,
		}
	}
	return item
}

// wikiOfficialsToDTO maps wiki official relations into the response format.
func wikiOfficialsToDTO(rels []dto.WikiOfficialRel) []dto.RatingOfficial {
	out := make([]dto.RatingOfficial, len(rels))
	for i, rel := range rels {
		alias := make([]string, len(rel.Official.Alias))
		for j, a := range rel.Official.Alias {
			alias[j] = a.Name
		}
		out[i] = dto.RatingOfficial{
			ID:       rel.Official.ID,
			Name:     rel.Official.Name,
			Link:     rel.Official.Link,
			Category: rel.Official.Category,
			Lang:     rel.Official.Lang,
			Alias:    alias,
		}
	}
	return out
}

// containsInt reports whether needle appears in haystack.
func containsInt(haystack []int, needle int) bool {
	if needle == 0 {
		return false
	}
	for _, v := range haystack {
		if v == needle {
			return true
		}
	}
	return false
}
