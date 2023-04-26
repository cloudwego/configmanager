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

package main

import (
	"log"
	"time"

	"github.com/cloudwego/configmanager"
	"github.com/cloudwego/configmanager/configvalue/items"
	"github.com/cloudwego/configmanager/fileprovider"
	"github.com/cloudwego/configmanager/iface"
	"github.com/cloudwego/configmanager/util"
)

var _ iface.ConfigKey = (*YourConfigKey)(nil)

// YourConfigKey is an example of a custom ConfigKey
type YourConfigKey struct {
	Service string
}

// ToString returns the string representation of the YourConfigKey instance
func (k *YourConfigKey) ToString() string {
	return k.Service
}

// NewYourConfigKey returns a new YourConfigKey
func NewYourConfigKey(s string) iface.ConfigKey {
	return &YourConfigKey{
		Service: s,
	}
}

func main() {
	// define your own item type
	const TypeItemMaxRetry iface.ItemType = "item-max-retry"
	const TypeItemMaxLimit iface.ItemType = "item-max-limit"
	const TypeItemServiceName iface.ItemType = "item-service-name"

	itemFactory := fileprovider.NewItemFactory(map[iface.ItemType]iface.ItemInitializer{
		TypeItemMaxRetry:    items.NewItemInt64,
		TypeItemMaxLimit:    items.NewItemInt64, // an item can be used with multiple item types
		TypeItemServiceName: items.NewItemString,
		// you can register your own ConfigValueItem(s) here
	})

	provider := fileprovider.NewFileProvider(
		fileprovider.WithFileProviderItemFactory(itemFactory),
		fileprovider.WithFileProviderPath("config.json"),
	)

	manager := configmanager.NewConfigManager([]configmanager.ConfigManagerOption{
		configmanager.WithRefreshInterval(10 * time.Second),
		configmanager.WithConfigProvider(provider),
	})

	manager.RegisterConfigChangeListener("unique-id-for-each-listener", func(change iface.ConfigChange) {
		// update something according to the change
		log.Println("Change:", string(util.MustJsonMarshal(change)))
	})

	manager.RefreshAndWait()

	configKey := NewYourConfigKey("test1")

	// it's recommended to retrieve the ConfigValueItem directly:
	maxRetryItem, err := manager.GetConfigItem(configKey, TypeItemMaxRetry)
	if err == nil {
		maxRetry := maxRetryItem.(*items.ItemInt64).Value()
		log.Println("max retry:", maxRetry)
	} else {
		log.Println("GetItem(maxRetry):", err)
	}

	// you can also retrieve the ConfigValueImpl for other purposes
	configValue, err := manager.GetConfig(configKey)
	if err == nil {
		maxRetryLimit, err := configValue.GetItem(TypeItemMaxLimit)
		if err == nil {
			maxRetry := maxRetryLimit.(*items.ItemInt64).Value()
			log.Println("max limit:", maxRetry)
		} else {
			log.Println("GetItem(maxLimit):", err)
		}

		// or
		serviceName := configValue.GetItemOrDefault(TypeItemServiceName, items.CopyDefaultItemString())
		if err == nil {
			log.Println("service name:", serviceName.(*items.ItemString).Value())
		} else {
			log.Println("GetItem(serviceName):", err)
		}
	} else {
		log.Println("GetConfig(configKey):", err)
	}

	err = manager.Dump("local_dump.json")
	if err != nil {
		log.Println("manager.Dump err:", err)
	} else {
		log.Println("manager.Dump to file succeeds.")
	}
}
