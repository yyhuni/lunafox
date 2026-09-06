package geolocation

import (
	"container/list"
	"time"
)

type cacheEntryKind uint8

const (
	cacheEntrySuccess cacheEntryKind = iota + 1
	cacheEntryNegative
)

type locationCacheEntry struct {
	ip             string
	kind           cacheEntryKind
	location       Location
	failureClass   FailureClass
	failedAt       time.Time
	nextEligibleAt time.Time
	expiresAt      time.Time
	recency        *list.Element
}

type locationCache struct {
	capacity    int
	successTTL  time.Duration
	negativeTTL time.Duration
	entries     map[string]*locationCacheEntry
	recency     *list.List
}

func newLocationCache(capacity int, successTTL, negativeTTL time.Duration) *locationCache {
	return &locationCache{
		capacity:    capacity,
		successTTL:  successTTL,
		negativeTTL: negativeTTL,
		entries:     make(map[string]*locationCacheEntry, capacity),
		recency:     list.New(),
	}
}

func (cache *locationCache) get(ip string, now time.Time) (*locationCacheEntry, bool) {
	entry, ok := cache.entries[ip]
	if !ok {
		return nil, false
	}
	if !now.Before(entry.expiresAt) {
		cache.remove(entry)
		return nil, false
	}
	cache.recency.MoveToFront(entry.recency)
	copy := *entry
	copy.location = cloneLocation(entry.location)
	copy.recency = nil
	return &copy, true
}

func (cache *locationCache) putSuccess(ip string, location Location, now time.Time) {
	cache.put(&locationCacheEntry{
		ip:        ip,
		kind:      cacheEntrySuccess,
		location:  cloneLocation(location),
		expiresAt: location.ResolvedAt.UTC().Add(cache.successTTL),
	}, now)
}

func (cache *locationCache) putNegative(ip string, failureClass FailureClass, failedAt time.Time) {
	failedAt = failedAt.UTC()
	cache.put(&locationCacheEntry{
		ip:             ip,
		kind:           cacheEntryNegative,
		failureClass:   failureClass,
		failedAt:       failedAt,
		nextEligibleAt: failedAt.Add(cache.negativeTTL),
		expiresAt:      failedAt.Add(cache.negativeTTL),
	}, failedAt)
}

func (cache *locationCache) put(entry *locationCacheEntry, now time.Time) {
	if existing, ok := cache.entries[entry.ip]; ok {
		cache.remove(existing)
	}
	cache.removeExpired(now)
	for len(cache.entries) >= cache.capacity {
		oldest := cache.recency.Back()
		if oldest == nil {
			break
		}
		cache.remove(oldest.Value.(*locationCacheEntry))
	}
	entry.recency = cache.recency.PushFront(entry)
	cache.entries[entry.ip] = entry
}

func (cache *locationCache) removeExpired(now time.Time) {
	for _, entry := range cache.entries {
		if !now.Before(entry.expiresAt) {
			cache.remove(entry)
		}
	}
}

func (cache *locationCache) remove(entry *locationCacheEntry) {
	delete(cache.entries, entry.ip)
	if entry.recency != nil {
		cache.recency.Remove(entry.recency)
		entry.recency = nil
	}
}

func cloneLocation(location Location) Location {
	copy := location
	if location.AccuracyRadiusKM != nil {
		radius := *location.AccuracyRadiusKM
		copy.AccuracyRadiusKM = &radius
	}
	return copy
}
