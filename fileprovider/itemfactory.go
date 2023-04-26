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

package fileprovider

import (
	"sync"

	"github.com/cloudwego/configmanager/iface"
)

// ItemFactory holds a map of item initializers with iface.ItemType as keys.
type ItemFactory struct {
	initializer *sync.Map // key: iface.ItemType, value: iface.ItemInitializer
}

// NewItemFactory returns a new ItemFactory with a given map of item initializers.
func NewItemFactory(given map[iface.ItemType]iface.ItemInitializer) *ItemFactory {
	initializers := &sync.Map{}
	for k, v := range given {
		initializers.Store(k, v)
	}
	return &ItemFactory{
		initializer: initializers,
	}
}

// NewItem returns a new instance of iface.ConfigValueItem with the given item type and json bytes.
func (f *ItemFactory) NewItem(itemType iface.ItemType, js []byte) (iface.ConfigValueItem, error) {
	if v, ok := f.initializer.Load(itemType); ok {
		initializer := v.(iface.ItemInitializer)
		return initializer(js)
	}
	return nil, iface.ErrNoItemInitializer
}

// RegisterInitializer registers an item initializer with the given item type.
func (f *ItemFactory) RegisterInitializer(itemType iface.ItemType, initializer iface.ItemInitializer) *ItemFactory {
	f.initializer.Store(itemType, initializer)
	return f
}
