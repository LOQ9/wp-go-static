package cache

import "sync"

// URLCache is a struct to hold the visited URLs
type URLCache struct {
	mu   sync.Mutex
	URLs map[string]bool
}

// Add adds a URL to the cache
func (c *URLCache) Add(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.URLs[url] = true
}

// Get checks if a URL is in the cache
func (c *URLCache) Get(url string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.URLs[url]
	return ok
}
