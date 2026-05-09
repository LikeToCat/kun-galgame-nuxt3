package service

import (
	"fmt"
	"strings"

	"kun-galgame-api/internal/galgame/dto"
	"kun-galgame-api/internal/galgame/model"
	"kun-galgame-api/internal/galgame/repository"
	"kun-galgame-api/internal/infrastructure/markdown"
	"kun-galgame-api/pkg/userclient"
)

// ──────────────────────────────────────────
// Shared slice/CSV utilities
// ──────────────────────────────────────────

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// groupResourceMeta bucketises rows from galgame_resource into per-galgame
// platform/language sets (preserving insertion order + dedup).
func groupResourceMeta(rows []model.GalgameResourceMeta) (platforms, languages map[int][]string) {
	platforms = make(map[int][]string)
	languages = make(map[int][]string)
	for _, r := range rows {
		if r.Platform != "" {
			platforms[r.GalgameID] = appendUniqueStr(platforms[r.GalgameID], r.Platform)
		}
		if r.Language != "" {
			languages[r.GalgameID] = appendUniqueStr(languages[r.GalgameID], r.Language)
		}
	}
	return
}

// ──────────────────────────────────────────
// Wiki → Detail DTO
// ──────────────────────────────────────────

// galgameDetailFromWiki maps a wiki galgame payload into the response DTO,
// resolving author/contributor users from the wiki-returned users map.
func galgameDetailFromWiki(g dto.WikiGalgameDetailFull, users map[string]dto.WikiUser) dto.GalgameDetail {
	return dto.GalgameDetail{
		ID:     g.ID,
		VndbID: g.VndbID,
		User:   lookupWikiUser(users, g.UserID),
		Name: dto.KunLanguage{
			EnUs: g.NameEnUs, JaJp: g.NameJaJp,
			ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
		},
		Banner: g.Banner,
		Introduction: dto.KunLanguage{
			EnUs: markdown.Render(g.IntroEnUs),
			JaJp: markdown.Render(g.IntroJaJp),
			ZhCn: markdown.Render(g.IntroZhCn),
			ZhTw: markdown.Render(g.IntroZhTw),
		},
		Markdown: dto.KunLanguage{
			EnUs: g.IntroEnUs, JaJp: g.IntroJaJp,
			ZhCn: g.IntroZhCn, ZhTw: g.IntroZhTw,
		},
		ContentLimit:       g.ContentLimit,
		ResourceUpdateTime: g.ResourceUpdateTime,
		OriginalLanguage:   g.OriginalLanguage,
		AgeLimit:           g.AgeLimit,
		Contributor:        contributorsFromWiki(g.Contributor, users),
		Alias:              wikiAliasesToNames(g.Alias),
		Engine:             enginesFromWiki(g.Engine),
		Official:           officialsFromWiki(g.Official),
		Tag:                tagsFromWiki(g.Tag),
		Created:            g.Created,
		Updated:            g.Updated,
	}
}

func lookupWikiUser(users map[string]dto.WikiUser, uid int) dto.UserBrief {
	if u, ok := users[fmt.Sprintf("%d", uid)]; ok {
		return dto.UserBrief{ID: u.ID, Name: u.Name, Avatar: u.Avatar}
	}
	return dto.UserBrief{ID: uid}
}

func contributorsFromWiki(contribs []dto.WikiContributor, users map[string]dto.WikiUser) []dto.UserBrief {
	out := make([]dto.UserBrief, len(contribs))
	for i, c := range contribs {
		out[i] = lookupWikiUser(users, c.UserID)
	}
	return out
}

func wikiAliasesToNames(aliases []dto.WikiAlias) []string {
	out := make([]string, len(aliases))
	for i, a := range aliases {
		out[i] = a.Name
	}
	return out
}

func enginesFromWiki(engines []dto.WikiEngineWithAlias) []dto.GalgameDetailEngine {
	out := make([]dto.GalgameDetailEngine, len(engines))
	for i, e := range engines {
		alias := e.Engine.Alias
		if alias == nil {
			alias = []string{}
		}
		out[i] = dto.GalgameDetailEngine{
			ID: e.Engine.ID, Name: e.Engine.Name, Alias: alias,
		}
	}
	return out
}

func officialsFromWiki(rels []dto.WikiOfficialRel) []dto.GalgameDetailOfficial {
	out := make([]dto.GalgameDetailOfficial, len(rels))
	for i, rel := range rels {
		out[i] = dto.GalgameDetailOfficial{
			ID:       rel.Official.ID,
			Name:     rel.Official.Name,
			Link:     rel.Official.Link,
			Category: rel.Official.Category,
			Lang:     rel.Official.Lang,
			Alias:    wikiAliasesToNames(rel.Official.Alias),
		}
	}
	return out
}

func tagsFromWiki(tags []dto.WikiTagWithSpoiler) []dto.GalgameDetailTag {
	out := make([]dto.GalgameDetailTag, len(tags))
	for i, t := range tags {
		out[i] = dto.GalgameDetailTag{
			ID:           t.Tag.ID,
			Name:         t.Tag.Name,
			Category:     t.Tag.Category,
			SpoilerLevel: t.SpoilerLevel,
		}
	}
	return out
}

// detailRatingFromRow maps a DB rating row into the detail-page rating card.
func detailRatingFromRow(
	r repository.GalgameDetailRatingRow,
	user userclient.User,
	isLiked bool,
	galgameID int,
	g dto.WikiGalgameDetailFull,
) dto.GalgameDetailRating {
	return dto.GalgameDetailRating{
		ID:           r.ID,
		User:         userBriefToDTO(user),
		Recommend:    r.Recommend,
		Overall:      r.Overall,
		View:         r.View,
		GalgameType:  rawJSON(r.GalgameType),
		PlayStatus:   r.PlayStatus,
		ShortSummary: r.ShortSummary,
		SpoilerLevel: r.SpoilerLevel,
		Art:          r.Art,
		Story:        r.Story,
		Music:        r.Music,
		Character:    r.Character,
		Route:        r.Route,
		System:       r.System,
		Voice:        r.Voice,
		ReplayValue:  r.ReplayValue,
		LikeCount:    r.LikeCount,
		IsLiked:      isLiked,
		GalgameID:    galgameID,
		Created:      r.Created,
		Updated:      r.Updated,
		Galgame: dto.GalgameDetailRatingGalgame{
			ID:           g.ID,
			ContentLimit: g.ContentLimit,
			Name: dto.KunLanguage{
				EnUs: g.NameEnUs, JaJp: g.NameJaJp,
				ZhCn: g.NameZhCn, ZhTw: g.NameZhTw,
			},
		},
	}
}

