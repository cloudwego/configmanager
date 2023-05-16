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

var (
	_             iface.ConfigValueItem = (*ItemBool)(nil)
	itemBoolTrue                        = ItemBool(true)
	itemBoolFalse                       = ItemBool(false)
)

// ItemBool is an alias of bool implementing iface.ConfigValueItem
type ItemBool bool

// NewItemBool decodes json bytes into a newly allocated ItemBool object
var NewItemBool = util.JsonInitializer(func() iface.ConfigValueItem {
	return itemBoolFalse.DeepCopy()
})

func CopyItemBoolTrue() iface.ConfigValueItem {
	return itemBoolTrue.DeepCopy()
}

func CopyItemBoolFalse() iface.ConfigValueItem {
	return itemBoolFalse.DeepCopy()
}

// DeepCopy creates a new copy of ItemBool implementing the ConfigValueItem interface.
func (i *ItemBool) DeepCopy() iface.ConfigValueItem {
	var n ItemBool = *i
	return &n
}

// EqualsTo determines if the ItemBool is equal to the given ConfigValueItem.
func (i *ItemBool) EqualsTo(item iface.ConfigValueItem) bool {
	o := item.(*ItemBool)
	return *i == *o
}

// Value returns the underlying bool value
func (i *ItemBool) Value() bool {
	return bool(*i)
}
