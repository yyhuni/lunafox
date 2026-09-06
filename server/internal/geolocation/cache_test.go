package geolocation

import (
	"testing"
	"time"
)

func TestLocationCacheUsesAbsoluteSuccessAndNegativeTTL(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cache := newLocationCache(4, 7*24*time.Hour, 5*time.Minute)
	radius := 12.5
	location := Location{
		ObservedIP:       "8.8.8.8",
		Latitude:         37.4219999,
		Longitude:        -122.0840575,
		AccuracyRadiusKM: &radius,
		ProviderKey:      FreeIPAPIProviderKey,
		ResolvedAt:       base,
	}

	cache.putSuccess(location.ObservedIP, location, base)
	entry, ok := cache.get(location.ObservedIP, base.Add(6*24*time.Hour))
	if !ok || entry.kind != cacheEntrySuccess || !entry.expiresAt.Equal(base.Add(7*24*time.Hour)) {
		t.Fatalf("success cache entry before expiry = %#v, found=%t", entry, ok)
	}
	if entry.location.AccuracyRadiusKM == location.AccuracyRadiusKM {
		t.Fatal("cache result must not expose the stored radius pointer")
	}
	if _, ok := cache.get(location.ObservedIP, base.Add(7*24*time.Hour)); ok {
		t.Fatal("success cache entry must expire at the exact seven-day boundary")
	}

	cache.putNegative("1.1.1.1", FailureNetwork, base)
	entry, ok = cache.get("1.1.1.1", base.Add(5*time.Minute-time.Nanosecond))
	if !ok || entry.kind != cacheEntryNegative || entry.failureClass != FailureNetwork || !entry.failedAt.Equal(base) || !entry.nextEligibleAt.Equal(base.Add(5*time.Minute)) {
		t.Fatalf("negative cache entry before expiry = %#v, found=%t", entry, ok)
	}
	if _, ok := cache.get("1.1.1.1", base.Add(5*time.Minute)); ok {
		t.Fatal("negative cache entry must expire at the exact five-minute boundary")
	}
}

func TestLocationCacheEvictsExpiredEntriesBeforeValidLRU(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cache := newLocationCache(2, 7*24*time.Hour, 5*time.Minute)
	cache.putSuccess("8.8.8.8", testLocation("8.8.8.8", base), base)
	cache.putSuccess("1.1.1.1", testLocation("1.1.1.1", base.Add(24*time.Hour)), base)

	cache.putSuccess("9.9.9.9", testLocation("9.9.9.9", base.Add(7*24*time.Hour)), base.Add(7*24*time.Hour))
	if _, ok := cache.entries["8.8.8.8"]; ok {
		t.Fatal("expired entry was retained when capacity was needed")
	}
	if _, ok := cache.entries["1.1.1.1"]; !ok {
		t.Fatal("valid entry was evicted before an expired entry")
	}
	if _, ok := cache.entries["9.9.9.9"]; !ok {
		t.Fatal("new entry was not cached")
	}
}

func TestLocationCacheUsesCombinedCapacityAndNonSlidingLRU(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	cache := newLocationCache(2, 7*24*time.Hour, 5*time.Minute)
	cache.putSuccess("8.8.8.8", testLocation("8.8.8.8", base), base)
	cache.putNegative("1.1.1.1", FailureNetwork, base)

	if _, ok := cache.get("8.8.8.8", base.Add(time.Minute)); !ok {
		t.Fatal("success entry unexpectedly missing")
	}
	cache.putSuccess("9.9.9.9", testLocation("9.9.9.9", base.Add(time.Minute)), base.Add(time.Minute))
	if _, ok := cache.entries["1.1.1.1"]; ok {
		t.Fatal("least-recently-used negative entry should share the same capacity and be evicted")
	}
	entry, ok := cache.get("8.8.8.8", base.Add(6*24*time.Hour))
	if !ok || !entry.expiresAt.Equal(base.Add(7*24*time.Hour)) {
		t.Fatalf("cache access extended absolute TTL: entry=%#v, found=%t", entry, ok)
	}
}

func TestLocationCacheRestartsEmpty(t *testing.T) {
	base := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	firstProcess := newLocationCache(2, 7*24*time.Hour, 5*time.Minute)
	firstProcess.putSuccess("8.8.8.8", testLocation("8.8.8.8", base), base)

	restartedProcess := newLocationCache(2, 7*24*time.Hour, 5*time.Minute)
	if _, ok := restartedProcess.get("8.8.8.8", base); ok || len(restartedProcess.entries) != 0 {
		t.Fatal("process-local cache must restart empty")
	}
}

func testLocation(ip string, resolvedAt time.Time) Location {
	return Location{
		ObservedIP:  ip,
		Latitude:    1,
		Longitude:   2,
		ProviderKey: FreeIPAPIProviderKey,
		ResolvedAt:  resolvedAt.UTC(),
	}
}
