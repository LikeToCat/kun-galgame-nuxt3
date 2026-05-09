package service

import (
	galgameClient "kun-galgame-api/internal/galgame/client"
	"kun-galgame-api/internal/user/dto"
	"kun-galgame-api/internal/user/repository"
)

// ──────────────────────────────────────────
// Shared helpers
// ──────────────────────────────────────────

func appendUniqueStr(slice []string, val string) []string {
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}

func emptyStrSlice(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func emptyLocale() dto.KunLanguage {
	return dto.KunLanguage{}
}

func briefToLocale(b galgameClient.GalgameBrief) dto.KunLanguage {
	return dto.KunLanguage{
		EnUs: b.NameEnUs, JaJp: b.NameJaJp,
		ZhCn: b.NameZhCn, ZhTw: b.NameZhTw,
	}
}

// groupResourceMeta bucketises (galgame_id, platform, language) tuples into
// per-galgame sets with insertion-order preservation and dedup.
func groupResourceMeta(rows []repository.GalgameResourceMeta) (platforms, languages map[int][]string) {
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

// collectUniqueIDs extracts unique int IDs from a slice via a projection.
func collectUniqueIDs[T any](rows []T, pick func(T) int) []int {
	out := make([]int, 0, len(rows))
	seen := make(map[int]bool, len(rows))
	for _, r := range rows {
		id := pick(r)
		if id > 0 && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
