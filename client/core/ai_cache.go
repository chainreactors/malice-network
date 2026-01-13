package core

import (
	"container/list"
	"strings"
	"sync"
	"time"
)

// AICompletionCache provides high-speed caching for AI completions.
// It supports:
// - exact match lookups
// - prefix-based reuse of cached results
// - TTL expiration
// - LRU eviction
type AICompletionCache struct {
	mu      sync.Mutex
	entries map[string]*list.Element
	lru     *list.List
	maxSize int
	ttl     time.Duration
}

type cacheEntry struct {
	Key         string
	Suggestions []string
	Timestamp   time.Time
}

const cacheScopeSeparator = "\x00"

// NewAICompletionCache creates a new cache with specified size and TTL
func NewAICompletionCache(maxSize int, ttl time.Duration) *AICompletionCache {
	return &AICompletionCache{
		entries: make(map[string]*list.Element),
		lru:     list.New(),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves cached completions for the given input
func (c *AICompletionCache) Get(input string) ([]string, bool) {
	key := normalizeCacheKey(input)
	if key == "" {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Try exact match first
	el, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	entry := el.Value.(*cacheEntry)
	if c.isExpired(entry) {
		c.removeElement(el)
		return nil, false
	}

	c.lru.MoveToFront(el)
	return cloneStringSlice(entry.Suggestions), true
}

// GetScoped retrieves cached completions for the given input under a namespace.
// Scope is typically the active menu name, so caches don't mix across contexts.
func (c *AICompletionCache) GetScoped(scope string, input string) ([]string, bool) {
	scopeKey := normalizeCacheKey(scope)
	if scopeKey == "" {
		return c.Get(input)
	}

	inputKey := normalizeCacheKey(input)
	if inputKey == "" {
		return nil, false
	}

	key := scopeKey + cacheScopeSeparator + inputKey

	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.entries[key]
	if !ok {
		return nil, false
	}

	entry := el.Value.(*cacheEntry)
	if c.isExpired(entry) {
		c.removeElement(el)
		return nil, false
	}

	c.lru.MoveToFront(el)
	return cloneStringSlice(entry.Suggestions), true
}

// GetPrefix retrieves cached completions matching the given prefix
func (c *AICompletionCache) GetPrefix(prefix string) ([]string, bool) {
	normalizedPrefix := normalizeCacheKey(prefix)
	if normalizedPrefix == "" {
		return nil, false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Find the longest matching prefix
	var bestEntry *cacheEntry
	var bestEl *list.Element

	for key, el := range c.entries {
		if !strings.HasPrefix(normalizedPrefix, key) {
			continue
		}

		entry := el.Value.(*cacheEntry)
		if c.isExpired(entry) {
			c.removeElement(el)
			continue
		}

		if bestEntry == nil || len(key) > len(bestEntry.Key) {
			bestEntry = entry
			bestEl = el
		}
	}

	if bestEntry == nil {
		return nil, false
	}

	filtered := make([]string, 0, len(bestEntry.Suggestions))
	for _, suggestion := range bestEntry.Suggestions {
		if strings.HasPrefix(strings.ToLower(suggestion), normalizedPrefix) {
			filtered = append(filtered, suggestion)
		}
	}
	if len(filtered) == 0 {
		return nil, false
	}

	c.lru.MoveToFront(bestEl)
	return filtered, true
}

// GetPrefixScoped retrieves cached completions matching the given prefix under a namespace.
func (c *AICompletionCache) GetPrefixScoped(scope string, prefix string) ([]string, bool) {
	scopeKey := normalizeCacheKey(scope)
	if scopeKey == "" {
		return c.GetPrefix(prefix)
	}

	normalizedPrefix := normalizeCacheKey(prefix)
	if normalizedPrefix == "" {
		return nil, false
	}

	scopePrefix := scopeKey + cacheScopeSeparator

	c.mu.Lock()
	defer c.mu.Unlock()

	var bestEntry *cacheEntry
	var bestEl *list.Element
	bestInputKeyLen := -1

	for key, el := range c.entries {
		if !strings.HasPrefix(key, scopePrefix) {
			continue
		}

		inputKey := strings.TrimPrefix(key, scopePrefix)
		if inputKey == "" {
			continue
		}

		if !strings.HasPrefix(normalizedPrefix, inputKey) {
			continue
		}

		entry := el.Value.(*cacheEntry)
		if c.isExpired(entry) {
			c.removeElement(el)
			continue
		}

		if bestEntry == nil || len(inputKey) > bestInputKeyLen {
			bestEntry = entry
			bestEl = el
			bestInputKeyLen = len(inputKey)
		}
	}

	if bestEntry == nil {
		return nil, false
	}

	filtered := make([]string, 0, len(bestEntry.Suggestions))
	for _, suggestion := range bestEntry.Suggestions {
		if strings.HasPrefix(strings.ToLower(suggestion), normalizedPrefix) {
			filtered = append(filtered, suggestion)
		}
	}
	if len(filtered) == 0 {
		return nil, false
	}

	c.lru.MoveToFront(bestEl)
	return filtered, true
}

// Set stores completions in the cache
func (c *AICompletionCache) Set(input string, suggestions []string) {
	key := normalizeCacheKey(input)
	if key == "" || len(suggestions) == 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.entries[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.Suggestions = cloneStringSlice(suggestions)
		entry.Timestamp = time.Now()
		c.lru.MoveToFront(el)
		return
	}

	entry := &cacheEntry{
		Key:         key,
		Suggestions: cloneStringSlice(suggestions),
		Timestamp:   time.Now(),
	}
	el := c.lru.PushFront(entry)
	c.entries[key] = el

	c.evictLRU()
}

// SetScoped stores completions in the cache under a namespace.
func (c *AICompletionCache) SetScoped(scope string, input string, suggestions []string) {
	scopeKey := normalizeCacheKey(scope)
	if scopeKey == "" {
		c.Set(input, suggestions)
		return
	}

	inputKey := normalizeCacheKey(input)
	if inputKey == "" || len(suggestions) == 0 {
		return
	}

	key := scopeKey + cacheScopeSeparator + inputKey

	c.mu.Lock()
	defer c.mu.Unlock()

	if el, ok := c.entries[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.Suggestions = cloneStringSlice(suggestions)
		entry.Timestamp = time.Now()
		c.lru.MoveToFront(el)
		return
	}

	entry := &cacheEntry{
		Key:         key,
		Suggestions: cloneStringSlice(suggestions),
		Timestamp:   time.Now(),
	}
	el := c.lru.PushFront(entry)
	c.entries[key] = el

	c.evictLRU()
}

func (c *AICompletionCache) evictLRU() {
	if c.maxSize <= 0 {
		return
	}

	for c.lru.Len() > c.maxSize {
		el := c.lru.Back()
		if el == nil {
			return
		}
		c.removeElement(el)
	}
}

func (c *AICompletionCache) removeElement(el *list.Element) {
	if el == nil {
		return
	}
	entry := el.Value.(*cacheEntry)
	delete(c.entries, entry.Key)
	c.lru.Remove(el)
}

func (c *AICompletionCache) isExpired(entry *cacheEntry) bool {
	if entry == nil {
		return true
	}
	if c.ttl <= 0 {
		return false
	}
	return time.Since(entry.Timestamp) >= c.ttl
}

// Clear removes all entries from the cache
func (c *AICompletionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*list.Element)
	c.lru.Init()
}

// Size returns the current number of entries
func (c *AICompletionCache) Size() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.entries)
}

// Cleanup removes expired entries
func (c *AICompletionCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ttl <= 0 {
		return
	}

	for el := c.lru.Back(); el != nil; {
		prev := el.Prev()
		entry := el.Value.(*cacheEntry)
		if time.Since(entry.Timestamp) >= c.ttl {
			c.removeElement(el)
		}
		el = prev
	}
}

func normalizeCacheKey(input string) string {
	return strings.TrimSpace(strings.ToLower(input))
}

func cloneStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
