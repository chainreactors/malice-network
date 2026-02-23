package core

import (
	"fmt"
	"github.com/chainreactors/IoM-go/consts"
	"github.com/chainreactors/IoM-go/proto/client/clientpb"
	"github.com/chainreactors/IoM-go/proto/implant/implantpb"
	"github.com/chainreactors/logs"
	"google.golang.org/protobuf/proto"
	"io"
	"math"
	"os"
	"sort"
	"sync"
	"time"
)

var CacheName = "cache.bin"

type Cache struct {
	items    sync.Map // map[string]*clientpb.SpiteCacheItem
	maxSize  int
	duration time.Duration
	savePath string
}

// NewCache initializes a new cache with a specified size, duration, and save path
func NewCache(savePath string) *Cache {
	cache := &Cache{
		maxSize:  1024,
		duration: math.MaxInt64,
		savePath: savePath,
	}
	GlobalTicker.Start(consts.DefaultCacheInterval, func() {
		err := cache.Save()
		if err != nil {
			logs.Log.Errorf("Failed to save cache: %v", err)
			return
		}
	})

	GlobalTicker.Start(consts.DefaultCacheInterval, cache.trim)
	return cache
}

// AddMessage adds a new item to the cache with TaskId and Index as part of the key
func (c *Cache) AddMessage(spite *implantpb.Spite, index int) {
	cacheKey := fmt.Sprintf("%d_%d", spite.TaskId, index)
	c.items.Store(cacheKey, &clientpb.SpiteCacheItem{
		Index:      int32(index),
		Id:         cacheKey,
		Spite:      spite,
		Expiration: time.Now().Add(c.duration).Unix(),
	})
}

// GetMessage retrieves a cache item by TaskId and Index
func (c *Cache) GetMessage(taskID, index int) (*implantpb.Spite, bool) {
	cacheKey := fmt.Sprintf("%d_%d", taskID, index)
	value, found := c.items.Load(cacheKey)
	if !found {
		return nil, false
	}
	item := value.(*clientpb.SpiteCacheItem)
	if time.Now().Unix() < item.Expiration {
		return item.Spite, true
	}
	return nil, false
}

func (c *Cache) GetMessages(taskID int) ([]*implantpb.Spite, bool) {
	type indexedSpite struct {
		index int32
		spite *implantpb.Spite
	}
	var items []indexedSpite
	now := time.Now().Unix()

	c.items.Range(func(key, value interface{}) bool {
		item := value.(*clientpb.SpiteCacheItem)
		if int(item.Spite.TaskId) == taskID && now < item.Expiration {
			items = append(items, indexedSpite{index: item.Index, spite: item.Spite})
		}
		return true
	})

	if len(items) == 0 {
		return nil, false
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].index < items[j].index
	})

	messages := make([]*implantpb.Spite, len(items))
	for i, item := range items {
		messages[i] = item.spite
	}
	return messages, true
}

// Save serializes the cache items to a file using protobuf
func (c *Cache) Save() error {
	file, err := os.Create(c.savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	items := &clientpb.SpiteCache{
		Items: make([]*clientpb.SpiteCacheItem, 0),
	}
	c.items.Range(func(key, value interface{}) bool {
		item := value.(*clientpb.SpiteCacheItem)
		items.Items = append(items.Items, item)
		return true
	})

	data, err := proto.Marshal(items)
	if err != nil {
		return err
	}
	if _, err = file.Write(data); err != nil {
		return err
	}
	return nil
}

// Load deserializes cache items from a file using protobuf
func (c *Cache) Load() error {
	file, err := os.OpenFile(c.savePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	item := &clientpb.SpiteCache{}
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if err = proto.Unmarshal(data, item); err != nil {
		return err
	}

	for _, item := range item.Items {
		c.items.Store(item.Id, item)
	}

	return nil
}

// trim removes expired items or items beyond max size
func (c *Cache) trim() {
	now := time.Now().Unix()

	// First pass: remove expired items
	c.items.Range(func(key, value interface{}) bool {
		item := value.(*clientpb.SpiteCacheItem)
		if now > item.Expiration {
			c.items.Delete(key)
		}
		return true
	})

	// Second pass: if still over maxSize, remove oldest items
	count := 0
	c.items.Range(func(key, value interface{}) bool {
		count++
		return true
	})

	if count > c.maxSize {
		// Collect all items with their expiration times
		type itemWithKey struct {
			key        interface{}
			expiration int64
		}
		var allItems []itemWithKey
		c.items.Range(func(key, value interface{}) bool {
			item := value.(*clientpb.SpiteCacheItem)
			allItems = append(allItems, itemWithKey{
				key:        key,
				expiration: item.Expiration,
			})
			return true
		})

		// Sort by expiration (oldest first) and delete excess
		toDelete := count - c.maxSize
		for i := 0; i < len(allItems) && toDelete > 0; i++ {
			oldestIdx := i
			for j := i + 1; j < len(allItems); j++ {
				if allItems[j].expiration < allItems[oldestIdx].expiration {
					oldestIdx = j
				}
			}
			if oldestIdx != i {
				allItems[i], allItems[oldestIdx] = allItems[oldestIdx], allItems[i]
			}
			c.items.Delete(allItems[i].key)
			toDelete--
		}
	}
}

// GetAll returns all items in the cache
func (c *Cache) GetAll() map[string]*implantpb.Spite {
	allItems := make(map[string]*implantpb.Spite)
	now := time.Now().Unix()
	c.items.Range(func(key, value interface{}) bool {
		item := value.(*clientpb.SpiteCacheItem)
		if now < item.Expiration {
			allItems[key.(string)] = item.Spite
		}
		return true
	})
	return allItems
}

func (c *Cache) GetLastMessage(taskID int) (*implantpb.Spite, bool) {
	var lastSpite *implantpb.Spite
	maxIndex := -1
	now := time.Now().Unix()

	c.items.Range(func(key, value interface{}) bool {
		item := value.(*clientpb.SpiteCacheItem)
		if int(item.Spite.TaskId) == taskID && int(item.Index) > maxIndex && now < item.Expiration {
			maxIndex = int(item.Index)
			lastSpite = item.Spite
		}
		return true
	})

	if lastSpite != nil {
		return lastSpite, true
	}
	return nil, false
}

type RingCache struct {
	capacity int
	messages []interface{}
}

func NewMessageCache(capacity int) *RingCache {
	return &RingCache{
		capacity: capacity,
		messages: make([]interface{}, 0, capacity),
	}
}

func (mc *RingCache) Add(message interface{}) {
	if len(mc.messages) == mc.capacity {
		mc.messages = mc.messages[1:]
	}
	// 添加新消息
	mc.messages = append(mc.messages, message)
}

func (mc *RingCache) GetAll() []interface{} {
	return mc.messages
}

func (mc *RingCache) GetLast() interface{} {
	if len(mc.messages) == 0 {
		return nil
	}
	return mc.messages[len(mc.messages)-1]
}

// GetN 返回最后 N 条消息
func (mc *RingCache) GetN(n int) []interface{} {
	if n <= 0 {
		return nil
	}
	if n > len(mc.messages) {
		n = len(mc.messages)
	}
	return mc.messages[len(mc.messages)-n:]
}
