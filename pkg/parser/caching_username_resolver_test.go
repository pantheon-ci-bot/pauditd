package parser

import (
	"github.com/stretchr/testify/assert"
	"os"
	"path"
	"sync"
	"testing"
	"time"
)

func Test_resolveCacheEnabled(t *testing.T) {
	cachedValues := make(map[string]string)
	resolver := &CachingUsernameResolver{
		cacheLock: &sync.Mutex{},
		cache:     cachedValues,
	}

	assert.Equal(t, "root", resolver.Resolve("0"), "0 should be root you animal")
	assert.Equal(t, "UNKNOWN_USER", resolver.Resolve("-1"), "Expected UNKNOWN_USER")

	val, ok := cachedValues["0"]
	if !ok {
		t.Fatal("Expected the uid mapping to be cached")
	}
	assert.Equal(t, "root", val)

	val, ok = cachedValues["-1"]
	if !ok {
		t.Fatal("Expected the uid mapping to be cached")
	}
	assert.Equal(t, "UNKNOWN_USER", val)
}

func Test_resolveNotCached(t *testing.T) {
	resolver := &DefaultUsernameResolver{}
	assert.Equal(t, "root", resolver.Resolve("0"), "0 should be root you animal")
	assert.Equal(t, "UNKNOWN_USER", resolver.Resolve("-1"), "Expected UNKNOWN_USER")
}

func Test_testCheckCache(t *testing.T) {
	filepath := path.Join(os.TempDir(), "test-passwd")
	f, _ := os.OpenFile(
		filepath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600,
	)
	defer f.Close()

	// setup cache
	resolver := &CachingUsernameResolver{
		cacheLock: &sync.Mutex{},
		cache: map[string]string{
			"0":   "notroot",
			"1":   "test2",
			"856": "test3",
		},
		lastFlush:  time.Now(),
		passwdPath: filepath,
	}

	// let time elapse
	time.Sleep(5 * time.Second)

	// test get cached value
	result := resolver.checkCache()
	assert.True(t, result)
	assert.Equal(t, 3, len(resolver.cache))

	// modify file
	f.Write([]byte("update write"))
	f.Sync()

	// test cache is cleared
	result = resolver.checkCache()
	assert.False(t, result)
	assert.Empty(t, resolver.cache)
}

func Test_rapid(t *testing.T) {
	filepath := path.Join(os.TempDir(), "test-passwd")
	f, _ := os.OpenFile(
		filepath,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600,
	)

	// setup test fixture file
	f.Write([]byte{})

	// prime cache with values for test
	resolver := &CachingUsernameResolver{
		cacheLock: &sync.Mutex{},
		cache: map[string]string{
			"0":   "notroot",
			"1":   "test2",
			"856": "test3",
		},
		lastFlush:  time.Now(),
		passwdPath: filepath,
	}

	tests := map[string]string{}
	fileModificationInterval := 4

	for i := 0; i < 3; i++ {
		go func() {
			var result string
			count := 0

			for input, expectedOutput := range tests {
				if count%fileModificationInterval == 0 {
					// modify file
					f.Write([]byte{})
				}
				// test method
				result = resolver.Resolve(input)
				assert.Equal(t, expectedOutput, result)
			}
		}()
	}
}

func Benchmark_getUsernameNoCache(b *testing.B) {
	resolver := NewDefaultUsernameResolver()
	for i := 0; i < b.N; i++ {
		_ = resolver.Resolve("0")
	}
}
func Benchmark_getUsernameCache(b *testing.B) {
	resolver := NewCachingUsernameResolver("")
	for i := 0; i < b.N; i++ {
		_ = resolver.Resolve("0")
	}
}
