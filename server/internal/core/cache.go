package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/patrickmn/go-cache"
	"os"
	"strconv"
	"strings"
)

type Cache struct {
	cache   cache.Cache
	maxSize int
}

func NewCache(maxSize int) *Cache {
	return &Cache{
		cache:   *cache.New(cache.NoExpiration, cache.NoExpiration),
		maxSize: maxSize,
	}
}

func (c *Cache) AddCache(spite *implantpb.Spite, cur int) {
	cacheKey := strconv.Itoa(int(spite.TaskId)) + "_" + strconv.Itoa(cur)
	c.cache.Set(cacheKey, spite, cache.NoExpiration)
	c.trim()
}

func (c *Cache) GetCache(taskID, cur int) (*implantpb.Spite, bool) {
	spite, found := c.cache.Get(strconv.Itoa(taskID) + "_" + strconv.Itoa(cur))
	return spite.(*implantpb.Spite), found
}

func (c *Cache) GetCaches(taskID int) ([]*implantpb.Spite, bool) {
	spite := make([]*implantpb.Spite, 0, c.cache.ItemCount())
	for k, v := range c.cache.Items() {
		parts := strings.Split(k, "_")
		if len(parts) != 2 {
			continue
		}
		taskIDStr := parts[0]
		if taskIDStr == strconv.Itoa(taskID) {
			spite = append(spite, v.Object.(*implantpb.Spite))
		}
	}
	return spite, true
}

func (c *Cache) Save(fileName string) error {
	err := c.cache.SaveFile(fileName)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Load(fileName string) error {
	_, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return fmt.Errorf("cache file %s does not exist", fileName)
	}
	err = c.cache.LoadFile(fileName)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) GetLastMessage(taskID int) (*implantpb.Spite, bool) {
	key := c.getMaxCurKeyForTaskID(taskID)
	value, success := c.cache.Get(key)
	return value.(*implantpb.Spite), success
}

func (c *Cache) SetSize(size int) {
	c.maxSize = size
	c.trim()
}

func (c *Cache) trim() {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	for _, v := range c.cache.Items() {
		gob.Register(v.Object)
	}
	err := enc.Encode(c.cache.Items())
	if err != nil {
		return
	}
	if buf.Len() > c.maxSize {
		c.cache.Delete(c.getMinTaskIDKey())
	}
}

func (c *Cache) getMinTaskIDKey() string {
	minKey := ""
	minTaskID := -1
	minCur := -1

	for key := range c.cache.Items() {
		parts := strings.Split(key, "_")
		if len(parts) != 2 {
			continue
		}
		taskIDStr := parts[0]
		curStr := parts[1]
		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			continue
		}
		cur, err := strconv.Atoi(curStr)
		if err != nil {
			continue
		}
		if minKey == "" || taskID < minTaskID || (taskID == minTaskID && cur < minCur) {
			minKey = key
			minTaskID = taskID
			minCur = cur
		}
	}
	return minKey
}

func (c *Cache) getMaxCurKeyForTaskID(taskID int) string {
	maxKey := ""
	maxCur := -1

	for key := range c.cache.Items() {
		parts := strings.Split(key, "_")
		if len(parts) != 2 {
			continue
		}
		taskIDStr := parts[0]
		curStr := parts[1]
		curTaskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			continue
		}
		if curTaskID != taskID {
			continue
		}
		cur, err := strconv.Atoi(curStr)
		if err != nil {
			continue
		}
		if maxKey == "" || cur > maxCur {
			maxKey = key
			maxCur = cur
		}
	}
	return maxKey
}
