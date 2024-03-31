package main

import (
	"InMemoryCache/internal"
	"time"
)

func main() {
	cache := internal.NewInMemoryCache(
		time.Duration(10), time.Duration(20),
	)

	cache.Set("server", "https://www.google.comd", time.Duration(20))
}
