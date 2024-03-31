package internal

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Cache interface {
	Get(key string) (interface{}, bool)
	Set(key string, value interface{}, duration time.Duration)
	Delete(key string) error
	Flush()
}

type Item struct {
	value      interface{}
	createdAt  time.Time
	expiration int64
}

type InMemoryCache struct {
	cache             map[string]Item
	rmu               sync.RWMutex
	defaultExpiration time.Duration
	cleanupInterval   time.Duration
}

func (c *InMemoryCache) Get(key string) (interface{}, bool) {
	c.rmu.RLock()
	defer c.rmu.RUnlock()
	item, found := c.cache[key]

	if !found {
		return nil, false
	}

	// Проверка на установку времени истечения, в противном случае он бессрочный
	if item.expiration > 0 {

		// Если в момент запроса кеш устарел возвращаем nil
		if time.Now().UnixNano() > item.expiration {
			return nil, false
		}

	}

	return item.value, true
}

func (c *InMemoryCache) Set(key string, value interface{}, duration time.Duration) {
	var expiration int64

	// Если продолжительность жизни равна 0 - используется значение по-умолчанию
	if duration == 0 {
		duration = c.defaultExpiration
	}

	// Устанавливаем время истечения кеша
	if duration > 0 {
		expiration = time.Now().Add(duration).UnixNano()
	}

	c.rmu.Lock()
	defer func() {
		fmt.Printf("Set key: %s value: %v expiration: %v\n", key, value, expiration)
		c.rmu.Unlock()
	}()

	c.cache[key] = Item{
		value:      value,
		createdAt:  time.Now(),
		expiration: expiration,
	}
}

func (c *InMemoryCache) Delete(key string) error {
	c.rmu.Lock()
	defer c.rmu.Unlock()

	if _, found := c.cache[key]; !found {
		errorString := "key: '" + key + "' not found"
		return errors.New(errorString)
	}

	delete(c.cache, key)
	return nil
}

func (c *InMemoryCache) StartGC() {
	go c.GC()
}

func (c *InMemoryCache) GC() {

	for {
		// ожидаем время установленное в cleanupInterval
		<-time.After(c.cleanupInterval)

		if c.cache == nil {
			return
		}

		// Ищем элементы с истекшим временем жизни и удаляем из хранилища
		if keys := c.expiredKeys(); len(keys) != 0 {
			c.clearItems(keys)
		}
	}
}

// expiredKeys возвращает список "просроченных" ключей
func (c *InMemoryCache) expiredKeys() (keys []string) {

	c.rmu.RLock()

	defer c.rmu.RUnlock()

	for k, i := range c.cache {
		if time.Now().UnixNano() > i.expiration && i.expiration > 0 {
			keys = append(keys, k)
		}
	}

	return
}

// clearItems удаляет ключи из переданного списка, в нашем случае "просроченные"
func (c *InMemoryCache) clearItems(keys []string) {
	fmt.Println("Clear items: ", keys)
	c.rmu.Lock()

	defer c.rmu.Unlock()

	for _, k := range keys {
		delete(c.cache, k)
	}
}

func (c *InMemoryCache) Flush() {
	c.rmu.Lock()
	defer c.rmu.Unlock()
	c.cache = make(map[string]Item)
}

func NewInMemoryCache(DefaultExpiration, CleanupInterval time.Duration) Cache {
	cache := &InMemoryCache{
		cache:             make(map[string]Item),
		defaultExpiration: DefaultExpiration,
		cleanupInterval:   CleanupInterval,
	}

	// Если интервал очистки больше 0, запускаем GC (удаление устаревших элементов)
	if CleanupInterval > 0 {
		cache.StartGC()
	}

	return cache
}
