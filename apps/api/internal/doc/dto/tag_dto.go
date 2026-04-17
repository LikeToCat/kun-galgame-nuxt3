package dto

// GetTagsRequest is the query for GET /doc/tag.
type GetTagsRequest struct {
	Page    int    `query:"page" validate:"min=1"`
	Limit   int    `query:"limit" validate:"min=1,max=50"`
	Keyword string `query:"keyword"`
}

// DeleteTagRequest is the query for DELETE /doc/tag.
type DeleteTagRequest struct {
	TagID int `query:"tagId" validate:"required,min=1"`
}
