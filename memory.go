// Copyright 2013 Beego Authors
// Copyright 2014 Unknwon
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package cache

import (
	"errors"
	"sync"
	"time"
)

// MemoryItem represents a memory cache item.
type MemoryItem struct {
	val     interface{}
	created int64
	expire  int64
}

// MemoryCacher represents a memory cache adapter implementation.
type MemoryCacher struct {
	lock     sync.RWMutex
	items    map[string]*MemoryItem
	interval int // GC interval.
}

// NewMemoryCacher creates and returns a new memory cacher.
func NewMemoryCacher() *MemoryCacher {
	return &MemoryCacher{items: make(map[string]*MemoryItem)}
}

// Put puts value into cache with key and expire time.
// If expired is 0, it will be deleted by next GC operation.
func (c *MemoryCacher) Put(key string, val interface{}, expire int64) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.items[key] = &MemoryItem{
		val:     val,
		created: time.Now().Unix(),
		expire:  expire,
	}
	return nil
}

// Get gets cached value by given key.
func (c *MemoryCacher) Get(key string) interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil
	}
	if item.expire > 0 &&
		(time.Now().Unix()-item.created) >= item.expire {
		go c.Delete(key)
		return nil
	}
	return item.val
}

// Delete deletes cached value by given key.
func (c *MemoryCacher) Delete(key string) error {
	c.lock.Lock()
	defer c.lock.Unlock()

	delete(c.items, key)
	return nil
}

// Incr increases cached int-type value by given key as a counter.
func (c *MemoryCacher) Incr(key string) (err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return errors.New("key not exist")
	}
	item.val, err = Incr(item.val)
	return err
}

// Decr decreases cached int-type value by given key as a counter.
func (c *MemoryCacher) Decr(key string) (err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return errors.New("key not exist")
	}

	item.val, err = Decr(item.val)
	return err
}

// IsExist returns true if cached value exists.
func (c *MemoryCacher) IsExist(key string) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()

	_, ok := c.items[key]
	return ok
}

// Flush deletes all cached data.
func (c *MemoryCacher) Flush() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.items = make(map[string]*MemoryItem)
	return nil
}

func (c *MemoryCacher) checkRawExpiration(key string) {
	item, ok := c.items[key]
	if !ok {
		return
	}

	if (time.Now().Unix() - item.created) >= item.expire {
		delete(c.items, key)
	}
}

func (c *MemoryCacher) checkExpiration(key string) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.checkRawExpiration(key)
}

func (c *MemoryCacher) startGC() {
	if c.interval < 1 {
		return
	}

	if c.items != nil {
		c.lock.Lock()
		defer c.lock.Unlock()
		for key, _ := range c.items {
			c.checkRawExpiration(key)
		}
	}

	time.AfterFunc(time.Duration(c.interval)*time.Second, func() { c.startGC() })
}

// StartAndGC starts GC routine based on config string settings.
func (c *MemoryCacher) StartAndGC(opt Options) error {
	c.interval = opt.Interval
	go c.startGC()
	return nil
}

func init() {
	Register("memory", NewMemoryCacher())
}
