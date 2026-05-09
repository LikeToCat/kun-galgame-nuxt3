// Package userclient is the HTTP SDK kungal uses to look up user identity
// from the OAuth server. Identity (name / avatar / bio / status / roles) is
// owned by OAuth post-migration; kungal stores only foreign keys (user_id)
// in business rows. Mapper layers call Users(ctx, ids) to enrich rows for
// rendering.
//
// Features:
//   - HTTP Basic Auth with OAuth client_id:client_secret (per OAuth doc § 10)
//   - In-memory TTL cache for hot users (default 10 min)
//   - Negative cache for not_found ids (default 1 min) to avoid repeat misses
//   - golang.org/x/sync/singleflight for in-flight dedup
//   - Auto-shard >100-id requests into 100-each chunks
//   - Ban-aware: status != 0 users get a placeholder name to keep render
//     paths from crashing while the caller decides whether to hide the row
package userclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// Config configures a Client. Caches are kept per Client instance — make it
// a singleton for cache to be effective.
type Config struct {
	BaseURL       string        // OAuth server, e.g. http://127.0.0.1:9277/api/v1
	ClientID      string        // OAuth Client (e.g. "kungal-backend")
	ClientSecret  string        // OAuth Client secret
	CacheTTL      time.Duration // hot cache TTL — defaults to 10 min
	NegCacheTTL   time.Duration // negative (not_found) TTL — defaults to 1 min
	HTTPTimeout   time.Duration // single-request timeout — defaults to 5 sec
	BatchPageSize int           // ids/request — defaults to 100 (OAuth max)
}

// User mirrors /users/batch's per-user payload.
type User struct {
	ID              int      `json:"id"`
	UUID            string   `json:"uuid"`
	Name            string   `json:"name"`
	Avatar          string   `json:"avatar"`
	AvatarImageHash string   `json:"avatar_image_hash"`
	Bio             string   `json:"bio"`
	Status          int      `json:"status"`
	Roles           []string `json:"roles"`
}

// Client is safe for concurrent use across goroutines.
type Client struct {
	cfg    Config
	http   *http.Client
	authHd string

	mu       sync.RWMutex
	hot      map[int]cacheEntry
	miss     map[int]time.Time
	sfGroup  singleflight.Group
	negTTL   time.Duration
	hotTTL   time.Duration
	pageSize int
}

type cacheEntry struct {
	user   User
	expire time.Time
}

// New returns a configured Client. Defaults fill missing fields.
func New(cfg Config) *Client {
	if cfg.CacheTTL == 0 {
		cfg.CacheTTL = 10 * time.Minute
	}
	if cfg.NegCacheTTL == 0 {
		cfg.NegCacheTTL = 1 * time.Minute
	}
	if cfg.HTTPTimeout == 0 {
		cfg.HTTPTimeout = 5 * time.Second
	}
	if cfg.BatchPageSize == 0 || cfg.BatchPageSize > 100 {
		cfg.BatchPageSize = 100
	}
	return &Client{
		cfg:      cfg,
		http:     &http.Client{Timeout: cfg.HTTPTimeout},
		authHd:   "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.ClientID+":"+cfg.ClientSecret)),
		hot:      map[int]cacheEntry{},
		miss:     map[int]time.Time{},
		hotTTL:   cfg.CacheTTL,
		negTTL:   cfg.NegCacheTTL,
		pageSize: cfg.BatchPageSize,
	}
}

// envelope is the OAuth API envelope shape `{code, message, data}`.
type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// batchData is the data field of /users/batch.
type batchData struct {
	Users    []User `json:"users"`
	NotFound []int  `json:"not_found"`
}

// searchData is the data field of /users/search.
type searchData struct {
	Items []User `json:"items"`
	Total int    `json:"total"`
}

// Users returns a map of id -> User. Missing IDs are absent (not nil entries).
// Order is irrelevant; caller looks up by id.
//
// The function transparently:
//   - serves from cache when fresh
//   - skips IDs in negative cache
//   - dedups concurrent requests for the same id via singleflight
//   - shards >100-id requests
//
// Returns the union of all known users; never returns an error solely because
// some ids are missing. A returned error means the OAuth call itself failed
// (network / 5xx / auth). On error, the partial cache hits are still in the map.
func (c *Client) Users(ctx context.Context, ids []int) (map[int]User, error) {
	out := map[int]User{}
	if len(ids) == 0 {
		return out, nil
	}

	// Dedup ids and pick out already-cached / known-missing.
	now := time.Now()
	seen := map[int]struct{}{}
	missing := make([]int, 0, len(ids))

	c.mu.RLock()
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if e, ok := c.hot[id]; ok && now.Before(e.expire) {
			out[id] = e.user
			continue
		}
		if t, ok := c.miss[id]; ok && now.Before(t) {
			continue // negative cache hit; skip
		}
		missing = append(missing, id)
	}
	c.mu.RUnlock()

	if len(missing) == 0 {
		return out, nil
	}

	// Shard remaining missing ids into pageSize chunks; fetch each with
	// singleflight keyed by the joined-id string so concurrent callers
	// asking for overlapping ids share work.
	for start := 0; start < len(missing); start += c.pageSize {
		end := start + c.pageSize
		if end > len(missing) {
			end = len(missing)
		}
		shard := missing[start:end]

		key := joinIntsForKey(shard)
		raw, err, _ := c.sfGroup.Do(key, func() (any, error) {
			return c.fetchBatch(ctx, shard)
		})
		if err != nil {
			return out, err
		}
		bd := raw.(batchData)
		c.cacheStore(bd, now)
		for _, u := range bd.Users {
			out[u.ID] = u
		}
	}
	return out, nil
}

// User is the single-id convenience. Returns (zero, false) for unknown ids.
func (c *Client) User(ctx context.Context, id int) (User, bool, error) {
	m, err := c.Users(ctx, []int{id})
	if err != nil {
		return User{}, false, err
	}
	u, ok := m[id]
	return u, ok, nil
}

// SearchUsers proxies OAuth /users/search. Results are not cached (search
// queries are too varied to cache effectively).
func (c *Client) SearchUsers(ctx context.Context, q string, page, limit int) ([]User, int, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 12
	}
	endpoint := c.cfg.BaseURL + "/users/search?" + url.Values{
		"q":     {q},
		"page":  {strconv.Itoa(page)},
		"limit": {strconv.Itoa(limit)},
	}.Encode()
	var sd searchData
	if err := c.do(ctx, "GET", endpoint, &sd); err != nil {
		return nil, 0, err
	}
	// Opportunistically warm the hot cache so a subsequent batch hit is free.
	now := time.Now()
	c.mu.Lock()
	for _, u := range sd.Items {
		c.hot[u.ID] = cacheEntry{user: u, expire: now.Add(c.hotTTL)}
	}
	c.mu.Unlock()
	return sd.Items, sd.Total, nil
}

// Placeholder builds a render-safe stub for users that are not_found or
// banned (status != 0). Mappers can call this so render paths stay
// non-nil even when an OAuth row is missing.
//
// id may be 0 to denote "unknown user" entirely.
func Placeholder(id int) User {
	return User{ID: id, Name: "已注销用户", Avatar: ""}
}

// fetchBatch makes a single /users/batch call with the given shard.
func (c *Client) fetchBatch(ctx context.Context, ids []int) (batchData, error) {
	endpoint := c.cfg.BaseURL + "/users/batch?ids=" + joinInts(ids, ",")
	var bd batchData
	if err := c.do(ctx, "GET", endpoint, &bd); err != nil {
		return bd, err
	}
	return bd, nil
}

// do runs an HTTP request, decodes the envelope, and unmarshal data into v.
// v should be a pointer.
func (c *Client) do(ctx context.Context, method, endpoint string, v any) error {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", c.authHd)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("userclient: %w", err)
	}
	defer resp.Body.Close()

	var env envelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("userclient: decode envelope: %w", err)
	}
	if env.Code != 0 {
		return fmt.Errorf("userclient: oauth code=%d msg=%q", env.Code, env.Message)
	}
	if v == nil {
		return nil
	}
	return json.Unmarshal(env.Data, v)
}

// cacheStore writes the batch result into hot + miss caches.
func (c *Client) cacheStore(bd batchData, now time.Time) {
	hotExp := now.Add(c.hotTTL)
	missExp := now.Add(c.negTTL)
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, u := range bd.Users {
		c.hot[u.ID] = cacheEntry{user: u, expire: hotExp}
		delete(c.miss, u.ID) // a previous miss was wrong, clear it
	}
	for _, id := range bd.NotFound {
		c.miss[id] = missExp
	}
}

// Invalidate drops a user id from hot+miss cache. Use after explicit
// updates (admin ban, etc.) so the next read goes back to OAuth.
func (c *Client) Invalidate(ids ...int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, id := range ids {
		delete(c.hot, id)
		delete(c.miss, id)
	}
}

func joinInts(xs []int, sep string) string {
	var b strings.Builder
	for i, x := range xs {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(strconv.Itoa(x))
	}
	return b.String()
}

func joinIntsForKey(xs []int) string {
	// Sort would be ideal for cache key normalization; chunks are short so
	// just use the joined string. Singleflight benefit only kicks in when
	// two concurrent callers happen to pass the same shard.
	return joinInts(xs, ",")
}
