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
	_ iface.ConfigValueItem = (*ItemString)(nil)

	defaultItemString = ItemString("")
)

// ItemString is an alias of string implementing iface.ConfigValueItem
type ItemString string

// CopyDefaultItemString returns a copy of the default configuration value item.
func CopyDefaultItemString() iface.ConfigValueItem {
	return defaultItemString.DeepCopy()
}

// ItemStringIface returns a ConfigValueItem with a string value.
func ItemStringIface(v string) iface.ConfigValueItem {
	i := ItemString(v)
	return &i
}

// NewItemString decodes json bytes into a newly allocated ItemString object
var NewItemString = util.JsonInitializer(func() iface.ConfigValueItem {
	return ItemStringIface("")
})

// DeepCopy creates a new copy of ItemString implementing the ConfigValueItem interface.
func (i *ItemString) DeepCopy() iface.ConfigValueItem {
	n := *i
	return &n
}

// EqualsTo determines if the ItemString is equal to the given ConfigValueItem.
func (i *ItemString) EqualsTo(item iface.ConfigValueItem) bool {
	o := item.(*ItemString)
	return *i == *o
}

// Value returns the underlying string value
func (i *ItemString) Value() string {
	return string(*i)
}
