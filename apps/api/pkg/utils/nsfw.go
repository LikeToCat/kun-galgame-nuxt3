package utils

import (
	"encoding/json"
	"net/url"

	"github.com/gofiber/fiber/v2"
)

// IsSFW reads the Pinia persisted settings cookie and returns
// whether the user has NSFW content disabled (default: true/SFW).
func IsSFW(c *fiber.Ctx) bool {
	raw := c.Cookies("KUNGalgameSettings", "")
	if raw == "" {
		return true
	}

	// Pinia persisted state may URL-encode the cookie value
	decoded, err := url.QueryUnescape(raw)
	if err != nil {
		decoded = raw
	}

	var settings struct {
		ShowKUNGalgameContentLimit string `json:"showKUNGalgameContentLimit"`
	}
	if err := json.Unmarshal([]byte(decoded), &settings); err != nil {
		return true
	}

	return settings.ShowKUNGalgameContentLimit != "nsfw"
}
