package service

import (
	"kun-galgame-api/internal/website/dto"
	"kun-galgame-api/internal/website/repository"
)

// websiteCardsFromRows maps slim website rows to WebsiteCard DTOs given the
// pre-loaded category-name and tag-level-sum maps.
//
// Both maps may be nil / empty; missing keys default to empty string / 0.
func websiteCardsFromRows(
	rows []repository.WebsiteListRow,
	catMap map[int]string,
	levelMap map[int]int,
) []dto.WebsiteCard {
	cards := make([]dto.WebsiteCard, len(rows))
	for i, r := range rows {
		lvl := levelMap[r.ID]
		cards[i] = dto.WebsiteCard{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Domain:      r.URL,
			AgeLimit:    r.AgeLimit,
			Level:       lvl,
			Icon:        r.Icon,
			Price:       lvl,
			Category:    catMap[r.CategoryID],
		}
	}
	return cards
}

// websiteCardsFromRowsSingleCategory is like websiteCardsFromRows but uses a
// fixed category name (used on the category detail endpoint where every row
// has the same category).
func websiteCardsFromRowsSingleCategory(
	rows []repository.WebsiteListRow,
	categoryName string,
	levelMap map[int]int,
) []dto.WebsiteCard {
	cards := make([]dto.WebsiteCard, len(rows))
	for i, r := range rows {
		lvl := levelMap[r.ID]
		cards[i] = dto.WebsiteCard{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
			Domain:      r.URL,
			AgeLimit:    r.AgeLimit,
			Level:       lvl,
			Icon:        r.Icon,
			Price:       lvl,
			Category:    categoryName,
		}
	}
	return cards
}

// collectCategoryIDs returns the unique category IDs from a slice of rows.
func collectCategoryIDs(rows []repository.WebsiteListRow) []int {
	ids := make([]int, 0, len(rows))
	seen := make(map[int]struct{}, len(rows))
	for _, r := range rows {
		if _, ok := seen[r.CategoryID]; ok {
			continue
		}
		seen[r.CategoryID] = struct{}{}
		ids = append(ids, r.CategoryID)
	}
	return ids
}

// collectWebsiteIDs returns the IDs from a slice of website rows.
func collectWebsiteIDs(rows []repository.WebsiteListRow) []int {
	ids := make([]int, len(rows))
	for i, r := range rows {
		ids[i] = r.ID
	}
	return ids
}
