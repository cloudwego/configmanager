// Copyright 2023 CloudWeGo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package configvalue

import (
	"encoding/json"
	"sync"
	"unsafe"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/configmanager/iface"
)

var (
	_ iface.ConfigValue = &ConfigValueImpl{}
	_ json.Marshaler    = &ConfigValueImpl{}
)

// ConfigValueImpl is a struct that represents a collection of configuration values
type ConfigValueImpl struct {
	lock  sync.RWMutex
	items map[iface.ItemType]iface.ConfigValueItem
}

// NewConfigValueImpl returns a new instance of ConfigValueImpl with the given items as input.
func NewConfigValueImpl(items map[iface.ItemType]iface.ConfigValueItem) *ConfigValueImpl {
	if items == nil {
		items = make(map[iface.ItemType]iface.ConfigValueItem)
	}
	return &ConfigValueImpl{
		items: items,
	}
}

// DeepCopy creates a deep copy of the ConfigValueImpl struct implementing the ConfigValue interface.
func (c *ConfigValueImpl) DeepCopy() iface.ConfigValue {
	c.lock.RLock()
	defer c.lock.RUnlock()
	v := &ConfigValueImpl{
		items: make(map[iface.ItemType]iface.ConfigValueItem, len(c.items)),
	}
	for k, item := range c.items {
		v.items[k] = item.DeepCopy()
	}
	return v
}

// EqualsTo checks whether the current ConfigValue is equal to the other ConfigValue.
func (c *ConfigValueImpl) EqualsTo(other iface.ConfigValue) bool {
	o, ok := other.(*ConfigValueImpl)
	if !ok {
		return false // should not happen
	}

	// lock with sequence to avoid deadlock
	first, second := determineLockSequence(c, o)
	first.lock.RLock()
	defer first.lock.RUnlock()
	second.lock.RLock()
	defer second.lock.RUnlock()

	if len(c.items) != len(o.items) {
		return false
	}
	for k, item := range c.items {
		if !item.EqualsTo(o.items[k]) {
			return false
		}
	}
	return true
}

func determineLockSequence(a, b *ConfigValueImpl) (first, second *ConfigValueImpl) {
	pa, pb := uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(b))
	if pa < pb {
		return a, b
	} else {
		return b, a
	}
}

// GetItem returns the item of the given type; if not found, returns iface.ErrItemNotFound
func (c *ConfigValueImpl) GetItem(itemType iface.ItemType) (iface.ConfigValueItem, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if c.items != nil {
		if item, ok := c.items[itemType]; ok {
			return item, nil
		}
	}
	return nil, iface.ErrItemNotFound
}

// GetItemOrDefault returns the item of a given type, or the default item if not found.
func (c *ConfigValueImpl) GetItemOrDefault(itemType iface.ItemType, defaultItem iface.ConfigValueItem) iface.ConfigValueItem {
	if item, err := c.GetItem(itemType); err == nil {
		return item
	}
	return defaultItem
}

// SetItem sets the value of a configuration item for a ConfigValueImpl instance.
func (c *ConfigValueImpl) SetItem(itemType iface.ItemType, item iface.ConfigValueItem) *ConfigValueImpl {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.items == nil {
		c.items = make(map[iface.ItemType]iface.ConfigValueItem)
	}
	c.items[itemType] = item
	return c
}

// MarshalJSON to implement the json.Marshaler interface.
func (c *ConfigValueImpl) MarshalJSON() ([]byte, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return sonic.ConfigStd.MarshalIndent(c.items, "", "\t")
}
