package internal

import (
	"context"
	"database/sql"
	"strings"
	"sync"
	"time"
)

// listCacheTTL bounds the age of the cached water lists. The cron job writes
// once per hour, and the fetch command can write from another process, so a
// short age limit keeps the lists current without a shared invalidation.
const listCacheTTL = 5 * time.Minute

// listCache keeps the unfiltered water list and the state list for each page.
// Search and state filters run on the cached list, so a request does not run
// the list query on SQLite.
type listCache struct {
	mu      sync.Mutex
	ttl     time.Duration
	entries map[bool]listEntry
}

type listEntry struct {
	waters   []Water
	states   []string
	loadedAt time.Time
}

func newListCache(ttl time.Duration) *listCache {
	return &listCache{ttl: ttl, entries: map[bool]listEntry{}}
}

// get returns the cached lists for the quality page or the temperature page.
// The lock is held during a load, so parallel requests start only one query.
func (c *listCache) get(ctx context.Context, db *sql.DB, quality bool) (listEntry, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[quality]; ok && time.Since(entry.loadedAt) < c.ttl {
		return entry, nil
	}

	source := "!" + agesSource
	list := ListWaters
	if quality {
		source = agesSource
		list = ListQualityWaters
	}
	waters, err := list(ctx, db, "", "")
	if err != nil {
		return listEntry{}, err
	}
	states, err := ListStates(ctx, db, source)
	if err != nil {
		return listEntry{}, err
	}
	entry := listEntry{waters: waters, states: states, loadedAt: time.Now()}
	c.entries[quality] = entry
	return entry, nil
}

// filterWaters applies the search and state filters. It returns a new slice
// and does not change the cached list.
func filterWaters(waters []Water, search, state string) []Water {
	if search == "" && state == "" {
		return waters
	}
	search = strings.ToLower(search)
	var filtered []Water
	for _, water := range waters {
		if state != "" && water.State != state {
			continue
		}
		if search != "" && !strings.Contains(strings.ToLower(water.Name), search) {
			continue
		}
		filtered = append(filtered, water)
	}
	return filtered
}
