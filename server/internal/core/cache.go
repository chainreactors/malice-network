package core

import (
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/helper/proto/client/clientpb"
	"github.com/chainreactors/malice-network/helper/proto/implant/implantpb"
	"google.golang.org/protobuf/proto"
	"io"
	"math"
	"os"
	"sync"
	"time"
)

var CacheName = "cache.bin"

type Cache struct {
	items    map[string]*clientpb.SpiteCacheItem
	mutex    sync.RWMutex
	maxSize  int
	duration time.Duration
	savePath string
}

// NewCache initializes a new cache with a specified size, duration, and save path
func NewCache(savePath string) *Cache {
	cache := &Cache{
		items:    make(map[string]*clientpb.SpiteCacheItem),
		maxSize:  1024,
		duration: math.MaxInt64,
		savePath: savePath + ".cache",
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	cacheKey := fmt.Sprintf("%d_%d", spite.TaskId, index)
	c.items[cacheKey] = &clientpb.SpiteCacheItem{
		Index:      int32(index),
		Id:         cacheKey,
		Spite:      spite,
		Expiration: time.Now().Add(c.duration).Unix(),
	}

	c.trim()
}

// GetMessage retrieves a cache item by TaskId and Index
func (c *Cache) GetMessage(taskID, index int) (*implantpb.Spite, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	cacheKey := fmt.Sprintf("%d_%d", taskID, index)
	item, found := c.items[cacheKey]
	if found && time.Now().Unix() < item.Expiration {
		return item.Spite, true
	}
	return nil, false
}

func (c *Cache) GetMessages(taskID int) ([]*implantpb.Spite, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var messages []*implantpb.Spite
	for _, item := range c.items {
		if int(item.Spite.TaskId) == taskID && time.Now().Unix() < item.Expiration {
			messages = append(messages, item.Spite)
		}
	}
	if len(messages) > 0 {
		return messages, true

	}
	return nil, false
}

// Save serializes the cache items to a file using protobuf
func (c *Cache) Save() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	file, err := os.Create(c.savePath)
	if err != nil {
		return err
	}
	defer file.Close()

	items := &clientpb.SpiteCache{
		Items: make([]*clientpb.SpiteCacheItem, 0, len(c.items)),
	}
	for _, item := range c.items {
		items.Items = append(items.Items, item)
	}
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
		c.items[item.Id] = item
	}

	return nil
}

// trim removes expired items or items beyond max size
func (c *Cache) trim() {
	for k, v := range c.items {
		if time.Now().Unix() > v.Expiration {
			delete(c.items, k)
		}
	}

	for len(c.items) > c.maxSize {
		var oldestKey string
		oldestTime := time.Now().Unix()
		for k, v := range c.items {
			if v.Expiration < oldestTime {
				oldestTime = v.Expiration
				oldestKey = k
			}
		}
		delete(c.items, oldestKey)
	}
}

// GetAll returns all items in the cache
func (c *Cache) GetAll() map[string]*implantpb.Spite {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	allItems := make(map[string]*implantpb.Spite)
	for k, v := range c.items {
		if time.Now().Unix() < v.Expiration {
			allItems[k] = v.Spite
		}
	}
	return allItems
}

func (c *Cache) GetLastMessage(taskID int) (*implantpb.Spite, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var lastSpite *implantpb.Spite
	maxIndex := -1

	for _, item := range c.items {
		if int(item.Spite.TaskId) == taskID && int(item.Index) > maxIndex && time.Now().Unix() < item.Expiration {
			maxIndex = int(item.Index)
			lastSpite = item.Spite
		}
	}

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
