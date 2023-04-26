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

const (
	// Int64Max is the maximum value of int64; exported for test cases
	Int64Max = 9223372036854775807

	// Int64MaxStr the maximum value of a signed 64-bit integer as a string constant.
	Int64MaxStr = "9223372036854775807"
)

var (
	_ iface.ConfigValueItem = (*ItemInt64)(nil)

	defaultItemInt64 = ItemInt64(Int64Max)
)

// ItemInt64 is an alias of int64 implementing iface.ConfigValueItem
type ItemInt64 int64

// CopyDefaultItemInt64 returns a copy of the default configuration value item.
func CopyDefaultItemInt64() iface.ConfigValueItem {
	return defaultItemInt64.DeepCopy()
}

// ItemInt64Iface returns a ConfigValueItem with an int64 value.
func ItemInt64Iface(v int64) iface.ConfigValueItem {
	i := ItemInt64(v)
	return &i
}

// NewItemInt64 decodes json bytes into a newly allocated ItemInt64 object
var NewItemInt64 = util.JsonInitializer(func() iface.ConfigValueItem {
	return ItemInt64Iface(0)
})

// DeepCopy creates a new copy of ItemInt64 implementing the ConfigValueItem interface.
func (i *ItemInt64) DeepCopy() iface.ConfigValueItem {
	var n ItemInt64 = *i
	return &n
}

// EqualsTo determines if the ItemInt64 is equal to the given ConfigValueItem.
func (i *ItemInt64) EqualsTo(item iface.ConfigValueItem) bool {
	o := item.(*ItemInt64)
	return *i == *o
}

// Value returns the int64 value of the ItemInt64.
func (i *ItemInt64) Value() int64 {
	return int64(*i)
}
