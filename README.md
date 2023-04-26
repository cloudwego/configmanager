# Config Manager

## Introduction

The `configmanager` is a package that provides a robust and efficient way to manage and
handle configuration data in your application. It's designed to load configurations
periodically from arbitrary sources, such as a file, a database, or a web service. The
module also provides monitoring on configuration changes and allows instant callback of
a registered listener.

## Features

* Refreshes configuration data at regular intervals specified
* Provides methods for accessing configuration data
* Supports configuration change listeners for efficient handling of updates
* Includes a default FileProvider for loading configurations from a file
* Allows for customizable options and arbitrary sources

## Overview

ConfigManager is designed with a 4-level hierarchy:

- ConfigManager:
  - Holds a `Provider` for loading of all configurations
  - Compares two versions of config and notify registered listeners if there's any difference
  - Provides APIs to reload config, retrieve certain config value/item, dump to files, etc.
- Provider:
  - Responsible for loading configurations from desired source(s)
  - Returns all configs in the form of `map[string]iface.ConfigValue` to the calling ConfigManager
- ConfigValue:
  - Holding a set of `iface.ConfigValueItem`
    - The builtin `ConfigValueImpl` is designed with a `map[iface.ItemType]iface.ConfigValueItem`
  - Provides APIs to retrieve certain config item
- ConfigValueItem:
  - Holds specific configurations according to requirement

## Usage

### Quick Start

> You can go directly to the `example/main.go` for a runnable example of the following steps.

`configmanager` ships with a `FileProvider` which loads from a config file in JSON format.

We'll begin with a JSON config file below (save it as `config.json`):

```
{
  "test1": {
    "item-max-retry": 1,
    "item-max-limit": 2,
    "item-service-name": "service_name"
  }
}
```

Import the package into your application.

```
import (
	"github.com/cloudwego/configmanager"
	"github.com/cloudwego/configmanager/configvalue/items"
	"github.com/cloudwego/configmanager/fileprovider"
	"github.com/cloudwego/configmanager/iface
)
```

Initialize a FileProvider
```
// define your own item type for builtin items
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
```

Create a new instance of ConfigManager with the desired options.
```
manager := configmanager.NewConfigManager([]configmanager.ConfigManagerOption{
    configmanager.WithRefreshInterval(10 * time.Second),
    configmanager.WithConfigProvider(provider),
)
```

Register configuration change listeners as needed:
```
manager.RegisterConfigChangeListener("unique-id-for-each-listener", func(change iface.ConfigChange) {
    // update something according to the change
    log.Println("Change:", string(util.MustJsonMarshal(change)))
})
```

Refresh the configuration data manually if necessary.
```
// send a refresh signal and return immediately:
manager.Refresh()

// or if you need to wait until the loading completes:
manager.RefreshAndWait()
```

Access and retrieve configuration using the provided methods:
```
// define your iface.ConfigKey with a ToString() method
configKey := NewYourConfigKey(...)

// it's recommended to retrieve the ConfigValueItem directly:
maxRetryItem, err := manager.GetConfigItem(configKey, TypeItemMaxRetry)
if err == nil {
    maxRetry := maxRetryItem.(*items.ItemInt64).Value()
    ...
}

// you can also retrieve the ConfigValueImpl for other purposes
configValue, err := manager.GetConfig(configKey)
if err == nil {
    maxRetryLimit, err := configValue.GetItem(TypeItemMaxLimit)
    // or call GetItemOrDefault() with a default item value
    serviceName := configValue.GetItemOrDefault(TypeItemServiceName, items.CopyDefaultItemString())
}
```

Export the configuration data to a file using the `Dump` method.
```
err := manager.Dump("local_dump.json")
```

### Customization

#### ConfigValueItems

Basic items like ItemBool, ItemInt64, ItemPair, ItemString is shipped with this package. You can
define your own ItemType for simple configuration only holding basic go type values with these items.

If you need items more complicated, just create your own items implementing the `iface.ConfigValueItem`.
It might be easier to copy from items.ItemPair and make modifications.

#### ConfigValue

The built-in `ConfigValueImpl` mainly focuses on its extensibility, which allows for extending of new
types of ConfigValueItem without modifying ConfigValueImpl and FileProvider, at the cost of increasing
its complexity (harder to comprehend).

If you wish, you can write your own implementation of `iface.ConfigValue`. It's not recommended, because
it's not compatible with the builtin FileProvider.

#### Provider

##### FileProvider

Available Options are:

* WithFileProviderPath
  * The specified file will be used to load the configuration data.
* WithFileProviderItemFactory
  * The ItemFactory is responsible for creating new instances of ConfigValueItem when loading data from the file.
* WithFileProviderLogger
  * This option sets a custom logger implementation for the FileProvider.
  * By default, the logger uses log.Printf.

##### Provider for other sources

You can implement your own Provider to load config from other sources, as long as it implements the
`iface.ConfigProvider`.

It's recommended to use `ConfigValueImpl` as the value in the returned config map, thus you can easily
switch your provider to `FileProvider`.

#### ConfigManager

Available Options are:

* WithRefreshInterval
  * This option sets the interval for refreshing the configuration data.
  * If not set, the interval defaults to 10 seconds.
* WithConfigProvider
  * This option sets the given ConfigProvider as the source for configuration data.
* WithErrorLogger
  * This option sets a custom error logger for the ConfigManager instance.
  * If not set, the logger defaults to log.Printf.
* WithConfigSerializer
  * This option sets a custom ConfigSerializer for the ConfigManager.
  * If not set, the serializer defaults to util.JSONSerializer.