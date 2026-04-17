package dto

import (
	"time"

	"kun-galgame-api/internal/website/model"
)

// ──────────────────────────────────────────
// Requests
// ──────────────────────────────────────────

type CreateWebsiteRequest struct {
	Name        string `json:"name" validate:"required,max=233"`
	URL         string `json:"url" validate:"required,url,max=500"`
	Description string `json:"description" validate:"max=1000"`
	Icon        string `json:"icon" validate:"max=500"`
	CategoryID  int    `json:"categoryId" validate:"required,min=1"`
	AgeLimit    string `json:"ageLimit" validate:"required,oneof=all r18"`
	Language    string `json:"language" validate:"max=10"`
	TagIDs      []int  `json:"tag_ids"`
}

type UpdateWebsiteRequest struct {
	WebsiteID   int    `json:"websiteId" validate:"required,min=1"`
	Name        string `json:"name" validate:"required,max=233"`
	URL         string `json:"url" validate:"required,url,max=500"`
	Description string `json:"description" validate:"max=1000"`
	Icon        string `json:"icon" validate:"max=500"`
	CategoryID  int    `json:"categoryId" validate:"required,min=1"`
	AgeLimit    string `json:"ageLimit" validate:"required,oneof=all r18"`
	Language    string `json:"language" validate:"max=10"`
	TagIDs      []int  `json:"tag_ids"`
}

type DeleteWebsiteRequest struct {
	WebsiteID int `query:"websiteId" validate:"required,min=1"`
}

type ToggleInteractionRequest struct {
	WebsiteID int `json:"websiteId" validate:"required,min=1"`
}

// ──────────────────────────────────────────
// Responses
// ──────────────────────────────────────────

// WebsiteCard is the list-card shape returned by GetWebsites and nested in
// category/tag detail responses.
type WebsiteCard struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	AgeLimit    string `json:"ageLimit"`
	Level       int    `json:"level"`
	Icon        string `json:"icon"`
	Price       int    `json:"price"`
	Category    string `json:"category"`
}

// WebsiteCategoryBrief is the category sub-object in the website detail response.
type WebsiteCategoryBrief struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

// WebsiteTagBrief is the tag sub-object in the website detail response.
type WebsiteTagBrief struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Label       string `json:"label"`
	Level       int    `json:"level"`
}

// UserBriefCompact is the user projection used for comments in the website detail.
type UserBriefCompact struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

// WebsiteDetailComment is a simple flat comment attached to the website detail.
type WebsiteDetailComment struct {
	ID      int              `json:"id"`
	Content string           `json:"content"`
	User    UserBriefCompact `json:"user"`
	Created string           `json:"created"`
	Updated string           `json:"updated"`
}

// WebsiteDetailResponse is the shape of GET /website/:domain.
// All field names and ordering are preserved verbatim from the original handler.
type WebsiteDetailResponse struct {
	ID            int                    `json:"id"`
	Name          string                 `json:"name"`
	URL           string                 `json:"url"`
	Description   string                 `json:"description"`
	Icon          string                 `json:"icon"`
	View          int                    `json:"view"`
	Language      string                 `json:"language"`
	AgeLimit      string                 `json:"ageLimit"`
	Category      WebsiteCategoryBrief   `json:"category"`
	Tags          []WebsiteTagBrief      `json:"tags"`
	LikeCount     int                    `json:"likeCount"`
	IsLiked       bool                   `json:"isLiked"`
	FavoriteCount int                    `json:"favoriteCount"`
	IsFavorited   bool                   `json:"isFavorited"`
	Domain        any                    `json:"domain"`
	CreateTime    string                 `json:"createTime"`
	Comment       []WebsiteDetailComment `json:"comment"`
	Created       time.Time              `json:"created"`
	Updated       time.Time              `json:"updated"`
}

// CreatedCommentResponse is the comment row returned by POST /website/:domain/comment.
type CreatedCommentResponse = model.GalgameWebsiteComment
