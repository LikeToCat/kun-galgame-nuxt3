package constants

// TopicSectionConsume lists sections that cost moemoepoints to post in.
var TopicSectionConsume = map[string]bool{
	"g-seeking": true,
	"g-other":   true,
	"t-help":    true,
}

// ValidTopicCategories are the allowed category values.
var ValidTopicCategories = []string{"galgame", "technique", "others"}

// ValidTopicSortFields are direct column sort fields.
var ValidTopicSortFields = map[string]string{
	"created":            "created",
	"view":               "view",
	"status_update_time": "status_update_time",
}

// ValidTopicCountSortFields map frontend sort names to count columns.
var ValidTopicCountSortFields = map[string]string{
	"like":     "like_count",
	"favorite": "favorite_count",
	"upvote":   "upvote_count",
}

const (
	MaxPollsPerTopic = 30
	MaxTagsPerTopic  = 7
	MaxTagLength     = 17
)
