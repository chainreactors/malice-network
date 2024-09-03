package core

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/chainreactors/logs"
	"github.com/chainreactors/malice-network/helper/consts"
	"github.com/chainreactors/malice-network/proto/implant/implantpb"
	"github.com/patrickmn/go-cache"
	"os"
	"strconv"
	"strings"
)

type Cache struct {
	cache    cache.Cache
	savePath string
	maxSize  int
}

func NewCache(maxSize int, savePath string) *Cache {
	newCache := &Cache{
		cache:    *cache.New(cache.NoExpiration, cache.NoExpiration),
		savePath: savePath,
		maxSize:  maxSize,
	}
	_, err := GlobalTicker.Start(consts.DefaultCacheJitter, func() {
		err := newCache.Save()
		if err != nil {
			logs.Log.Errorf("save cache error %s", err.Error())
		}
	})
	if err != nil {
		return nil
	}
	return newCache
}

func (c *Cache) AddMessage(spite *implantpb.Spite, index int) {
	cacheKey := strconv.Itoa(int(spite.TaskId)) + "_" + strconv.Itoa(index)
	c.cache.Set(cacheKey, spite, cache.NoExpiration)
	c.trim()
}

func (c *Cache) GetMessage(taskID, index int) (*implantpb.Spite, bool) {
	spite, found := c.cache.Get(fmt.Sprintf("%d_%d", taskID, index))
	if found {
		return spite.(*implantpb.Spite), found
	} else {
		return nil, false
	}
}

func (c *Cache) GetMessages(taskID int) ([]*implantpb.Spite, bool) {
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

//func (c *Cache) GetAll()  {
//	for k, v := range c.cache.Items() {
//		logs.Log.Importantf(k, v)
//	}
//}

func (c *Cache) Save() error {
	err := c.cache.SaveFile(c.savePath)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) Load() error {
	gob.Register(&implantpb.Spite{})
	_, err := os.Stat(c.savePath)
	if os.IsNotExist(err) {
		return fmt.Errorf("cache file %s does not exist", c.savePath)
	}
	err = c.cache.LoadFile(c.savePath)
	if err != nil {
		return err
	}
	return nil
}

func (c *Cache) GetLastMessage(taskID int) (*implantpb.Spite, bool) {
	key := c.getMaxCurKeyForTaskID(taskID)
	value, ok := c.cache.Get(key)
	if ok {
		return value.(*implantpb.Spite), ok
	}
	return nil, false
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
