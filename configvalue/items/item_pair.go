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

package items

import (
	"github.com/cloudwego/configmanager/iface"
	"github.com/cloudwego/configmanager/util"
)

var _ iface.ConfigValueItem = &ItemPair{}

var defaultItemPair = &ItemPair{
	Key:   "",
	Value: "",
}

func CopyDefaultItemPair() iface.ConfigValueItem {
	return defaultItemPair.DeepCopy()
}

// ItemPair is a struct that represents a key-value pair
type ItemPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewItemPair decodes json bytes into a new ItemPair object
var NewItemPair = util.JsonInitializer(func() iface.ConfigValueItem {
	return &ItemPair{}
})

// DeepCopy creates a deep copy of the ItemPair struct to eliminate any references to the original data.
func (i *ItemPair) DeepCopy() iface.ConfigValueItem {
	return &ItemPair{
		Key:   i.Key,
		Value: i.Value,
	}
}

// EqualsTo checks if the current ItemPair is equal to the given ConfigValueItem.
func (i *ItemPair) EqualsTo(item iface.ConfigValueItem) bool {
	o := item.(*ItemPair)
	return i.Key == o.Key && i.Value == o.Value
}
