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
	"crypto/md5"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/cloudwego/configmanager/configvalue"
	"github.com/cloudwego/configmanager/iface"
	"github.com/cloudwego/configmanager/util"
)

var _ iface.ConfigProvider = &FileProvider{}

// FileProvider loads config from a file.
type FileProvider struct {
	filepath    string
	itemFactory *ItemFactory
	logger      iface.LogFunc
	logLimiter  iface.Limiter
	lastMD5sum  [md5.Size]byte
}

// FileProviderOption is used to control the behavior of FileProvider
type FileProviderOption func(p *FileProvider)

// NewFileProvider creates a new file provider with the specified options.
func NewFileProvider(options ...FileProviderOption) *FileProvider {
	p := &FileProvider{
		logger: log.Printf,
	}

	for _, option := range options {
		option(p)
	}

	if p.itemFactory == nil {
		panic("invalid item factory")
	}

	if p.filepath == "" {
		panic("file path not set")
	}
	return p
}

// WithFileProviderLogger sets the logger implementation for the file provider option.
func WithFileProviderLogger(logger iface.LogFunc) FileProviderOption {
	return func(p *FileProvider) {
		p.logger = logger
	}
}

// WithFileProviderPath sets the file path for the file provider option.
func WithFileProviderPath(filepath string) FileProviderOption {
	return func(p *FileProvider) {
		p.filepath = filepath
	}
}

// WithFileProviderItemFactory returns a FileProviderOption that sets the ItemFactory.
func WithFileProviderItemFactory(itemFactory *ItemFactory) FileProviderOption {
	return func(p *FileProvider) {
		p.itemFactory = itemFactory
	}
}

// WithLogLimiter returns a FileProviderOption that sets the log limiter.
func WithLogLimiter(limiter iface.Limiter) FileProviderOption {
	return func(p *FileProvider) {
		p.logLimiter = limiter
	}
}

// log logs the message when the limiter allows.
func (p *FileProvider) log(format string, args ...interface{}) {
	if p.logger == nil {
		return
	}
	if p.logLimiter == nil || p.logLimiter.Allow() {
		p.logger(format, args...)
	}
}

// LoadConfig loads configuration for FileProvider
func (p *FileProvider) LoadConfig(currentConfig map[string]iface.ConfigValue) (map[string]iface.ConfigValue, error) {
	contents, err := p.readFile()
	if err != nil {
		return nil, err
	}

	md5sum := md5.Sum(contents)
	if p.lastMD5sum == md5sum {
		return nil, iface.ErrNotModified
	} else {
		p.lastMD5sum = md5sum
	}

	configMap := make(map[string]map[string]json.RawMessage)
	if err = util.JsonDecodeWithNumber(contents, &configMap); err != nil {
		p.log("FileProvider unmarshal file failed: file=%s, err=%v", p.filepath, err)
		return nil, err
	}

	result := make(map[string]iface.ConfigValue)
	for key, value := range configMap {
		result[key] = p.convertToConfigValueImpl(key, value)
	}
	return result, nil
}

func (p *FileProvider) readFile() ([]byte, error) {
	file, err := os.Open(p.filepath)
	if err != nil {
		p.log("FileProvider open file failed: file=%s, err=%v", p.filepath, err)
		return nil, err
	}
	defer file.Close()

	contents, err := ioutil.ReadAll(file)
	if err != nil {
		p.log("FileProvider read file failed: file=%s, err=%v", p.filepath, err)
		return nil, err
	}
	return contents, nil
}

// Since items in configValue have different types, it's impossible to unmarshal directly from json.
// Here we marshal each item to json first and then unmarshal to the object allocated by ItemFactory.
func (p *FileProvider) convertToConfigValueImpl(configKey string, configValue map[string]json.RawMessage) *configvalue.ConfigValueImpl {
	configValueImpl := configvalue.NewConfigValueImpl(nil)
	for itemKey, itemValue := range configValue {
		itemInstance, err := p.itemFactory.NewItem(iface.ItemType(itemKey), itemValue)
		if err != nil {
			p.log("FileProvider load config failed when new item: configKey=%s, err=%v, json=%s", configKey, err, itemValue)
			continue
		}
		configValueImpl.SetItem(iface.ItemType(itemKey), itemInstance)
	}
	return configValueImpl
}
