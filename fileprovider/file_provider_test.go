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
	"encoding/json"
	"errors"
	"io"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/configmanager/configvalue/items"

	"github.com/cloudwego/configmanager/configvalue"
	"github.com/cloudwego/configmanager/iface"
	"github.com/cloudwego/configmanager/util"
)

func TestWithFileProviderLogger(t *testing.T) {
	type args struct {
		logger iface.LogFunc
	}
	logger := func(format string, v ...interface{}) {}
	tests := []struct {
		name string
		args args
		isOK func(p *FileProvider) bool
	}{
		{
			name: "nil-logger",
			args: args{
				logger: nil,
			},
			isOK: func(p *FileProvider) bool {
				return p.logger == nil
			},
		},
		{
			name: "not-nil-logger",
			args: args{
				logger: logger,
			},
			isOK: func(p *FileProvider) bool {
				return reflect.ValueOf(p.logger).Pointer() == reflect.ValueOf(logger).Pointer()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := []FileProviderOption{
				WithFileProviderPath("test"),
				WithFileProviderLogger(tt.args.logger),
				WithFileProviderItemFactory(NewItemFactory(nil)),
			}
			if got := NewFileProvider(options...); !tt.isOK(got) {
				t.Errorf("NewFileProvider() = %v", got)
			}
		})
	}
}

func TestWithFileProviderPath(t *testing.T) {
	type args struct {
		filepath string
	}
	tests := []struct {
		name      string
		args      args
		willPanic bool
		isOK      func(p *FileProvider) bool
	}{
		{
			name: "empty-path",
			args: args{
				filepath: "",
			},
			willPanic: true,
			isOK: func(p *FileProvider) bool {
				return p.filepath == ""
			},
		},
		{
			name: "not-empty-path",
			args: args{
				filepath: "test",
			},
			willPanic: false,
			isOK: func(p *FileProvider) bool {
				return p.filepath == "test"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.willPanic {
					t.Errorf("NewFileProvider() panic = %v", r)
				}
			}()
			if got := NewFileProvider(WithFileProviderPath(tt.args.filepath), WithFileProviderItemFactory(NewItemFactory(nil))); !tt.isOK(got) {
				t.Errorf("NewFileProvider() = %+v", got)
			}
		})
	}
}

func TestWithFileProviderItemFactory(t *testing.T) {
	type args struct {
		itemFactory *ItemFactory
	}
	tests := []struct {
		name      string
		args      args
		willPanic bool
		isOK      func(p *FileProvider) bool
	}{
		{
			name: "nil-factory",
			args: args{
				itemFactory: nil,
			},
			willPanic: true,
			isOK: func(p *FileProvider) bool {
				return p.itemFactory == nil
			},
		},
		{
			name: "not-nil-factory",
			args: args{
				itemFactory: NewItemFactory(nil),
			},
			willPanic: false,
			isOK: func(p *FileProvider) bool {
				return p.itemFactory != nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil && !tt.willPanic {
					t.Errorf("NewFileProvider() panic = %v", r)
				}
			}()
			if got := NewFileProvider(WithFileProviderPath("test.json"), WithFileProviderItemFactory(tt.args.itemFactory)); !tt.isOK(got) {
				t.Errorf("NewFileProvider() = %v", got)
			}
		})
	}
}

func TestWithLogLimiter(t *testing.T) {
	t.Run("no-limiter", func(t *testing.T) {
		counter := int32(0)
		limiter := util.NewFrequencyLimiter(time.Second, 1)
		defer limiter.Stop()
		p := NewFileProvider(
			WithFileProviderPath("test.json"),
			WithFileProviderItemFactory(NewItemFactory(nil)),
			WithFileProviderLogger(func(format string, v ...interface{}) {
				atomic.AddInt32(&counter, 1)
			}),
		)
		p.log("test")
		p.log("test")
		if atomic.LoadInt32(&counter) != 2 {
			t.Errorf("log() should not be limited, want 2")
		}
	})
	t.Run("with-limiter", func(t *testing.T) {
		counter := int32(0)
		limiter := util.NewFrequencyLimiter(time.Second, 1)
		defer limiter.Stop()
		p := NewFileProvider(
			WithFileProviderPath("test.json"),
			WithFileProviderItemFactory(NewItemFactory(nil)),
			WithFileProviderLogger(func(format string, v ...interface{}) {
				atomic.AddInt32(&counter, 1)
			}),
			WithLogLimiter(limiter),
		)
		p.log("test")
		p.log("test")
		if atomic.LoadInt32(&counter) != 1 {
			t.Errorf("log() should be limited")
		}
	})
}

func TestFileProvider_readFile(t *testing.T) {
	type fields struct {
		contents    string
		itemFactory *ItemFactory
		logger      iface.LogFunc
	}
	filePath := "test_file_provider.tmp"
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name: "empty-file",
			fields: fields{
				contents: "",
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "not-empty-file",
			fields: fields{
				contents: "test",
			},
			want:    "test",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := writeFile(filePath, tt.fields.contents); err != nil {
				t.Errorf("write file error: %v", err)
				return
			}

			p := &FileProvider{
				filepath:    filePath,
				itemFactory: tt.fields.itemFactory,
				logger:      tt.fields.logger,
			}

			got, err := p.readFile()
			if (err != nil) != tt.wantErr {
				t.Errorf("readFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, []byte(tt.want)) {
				t.Errorf("readFile() got = %v, want %v", got, tt.want)
			}

			if err := os.Remove(filePath); err != nil {
				t.Errorf("remove file error: %v", err)
			}
		})
	}
}

func writeFile(filePath, contents string) error {
	var f *os.File
	var err error

	if f, err = os.Create(filePath); err != nil {
		return err
	}
	if _, err = io.WriteString(f, contents); err != nil {
		return err
	}
	return f.Close()
}

func TestFileProvider_convertToConfigValueImpl(t *testing.T) {
	type fields struct {
		itemFactory *ItemFactory
	}
	type args struct {
		configKey   string
		configValue map[string]json.RawMessage
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *configvalue.ConfigValueImpl
	}{
		{
			name: "config-value-with-item-int-unknown",
			fields: fields{
				itemFactory: NewItemFactory(nil),
			},
			args: args{
				configKey: "test",
				configValue: map[string]json.RawMessage{
					string(itemTypeInt): []byte(`{"value": 9223372036854775807}`),
				},
			},
			want: configvalue.NewConfigValueImpl(nil),
		},
		{
			name: "config-value-with-item-int-with-factory",
			fields: fields{
				itemFactory: NewItemFactory(nil).RegisterInitializer(itemTypeInt, NewItemInt),
			},
			args: args{
				configKey: "test",
				configValue: map[string]json.RawMessage{
					string(itemTypeInt): []byte(`{"value": 9223372036854775807}`),
				},
			},
			want: configvalue.NewConfigValueImpl(map[iface.ItemType]iface.ConfigValueItem{
				itemTypeInt: &ItemInt{
					Value: items.Int64Max,
				},
			}),
		},
		{
			name: "config-value-with-item-int-and-pair",
			fields: fields{
				itemFactory: NewItemFactory(map[iface.ItemType]iface.ItemInitializer{
					itemTypeInt:  NewItemInt,
					itemTypePair: items.NewItemPair,
				}),
			},
			args: args{
				configKey: "test",
				configValue: map[string]json.RawMessage{
					string(itemTypeInt):  []byte(`{"value": 9223372036854775807}`),
					string(itemTypePair): []byte(`{"key": "key", "value": "value"}`),
				},
			},
			want: configvalue.NewConfigValueImpl(map[iface.ItemType]iface.ConfigValueItem{
				itemTypeInt: &ItemInt{
					Value: items.Int64Max,
				},
				itemTypePair: &items.ItemPair{
					Key:   "key",
					Value: "value",
				},
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewFileProvider(
				WithFileProviderPath("test.json"),
				WithFileProviderItemFactory(tt.fields.itemFactory),
			)
			if got := p.convertToConfigValueImpl(tt.args.configKey, tt.args.configValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToConfigValueImpl() = %v, want %v", string(util.MustJsonMarshal(got)), string(util.MustJsonMarshal(tt.want)))
			}
		})
	}
}

func TestFileProvider_LoadConfig(t *testing.T) {
	type fields struct {
		fileContent []byte
		itemFactory *ItemFactory
	}
	type args struct {
		currentConfig map[string]iface.ConfigValue
		callTwice     bool
	}
	filePath := "test_file_provider.tmp"
	tests := []struct {
		name         string
		fields       fields
		args         args
		want         map[string]iface.ConfigValue
		wantErr      bool
		wantErrValue error
	}{
		{
			name: "load-item-int-and-item-pair",
			fields: fields{
				fileContent: util.MustJsonMarshal(map[string]interface{}{
					"test": map[string]interface{}{
						string(itemTypeInt): json.Number(items.Int64MaxStr),
						string(itemTypePair): map[string]interface{}{
							"key":   "key",
							"value": "value",
						},
						string(itemTypeAllowXXX): true,
						string(itemTypeDenyXXX):  false,
						string(itemTypeString):   "hello",
					},
				}),
				itemFactory: NewItemFactory(map[iface.ItemType]iface.ItemInitializer{
					itemTypeInt:      items.NewItemInt64,
					itemTypePair:     items.NewItemPair,
					itemTypeAllowXXX: items.NewItemBool,
					itemTypeDenyXXX:  items.NewItemBool,
					itemTypeString:   items.NewItemString,
				}),
			},
			args: args{
				currentConfig: map[string]iface.ConfigValue{},
			},
			want: map[string]iface.ConfigValue{
				"test": configvalue.NewConfigValueImpl(map[iface.ItemType]iface.ConfigValueItem{
					itemTypeInt: items.ItemInt64Iface(items.Int64Max),
					itemTypePair: &items.ItemPair{
						Key:   "key",
						Value: "value",
					},
					itemTypeAllowXXX: items.CopyItemBoolTrue(),
					itemTypeDenyXXX:  items.CopyItemBoolFalse(),
					itemTypeString:   items.ItemStringIface("hello"),
				}),
			},
		},
		{
			name: "load-item-twice-with-file-no-change",
			fields: fields{
				fileContent: util.MustJsonMarshal(map[string]interface{}{}),
				itemFactory: NewItemFactory(nil),
			},
			args: args{
				currentConfig: map[string]iface.ConfigValue{},
				callTwice:     true,
			},
			want:         map[string]iface.ConfigValue{},
			wantErr:      true,
			wantErrValue: iface.ErrNotModified,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := writeFile(filePath, string(tt.fields.fileContent)); err != nil {
				t.Errorf("write file failed: %v", err)
				return
			}
			defer func() {
				if err := os.Remove(filePath); err != nil {
					t.Errorf("remove file failed: %v", err)
				}
			}()
			p := NewFileProvider(
				WithFileProviderPath(filePath),
				WithFileProviderItemFactory(tt.fields.itemFactory),
			)
			got, err := p.LoadConfig(tt.args.currentConfig)
			if tt.args.callTwice {
				got, err = p.LoadConfig(tt.args.currentConfig)
			}
			if tt.wantErr {
				if err == nil {
					t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErrValue)
					return
				}
				if !errors.Is(err, tt.wantErrValue) {
					t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErrValue)
					return
				}
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadConfig() got = %v, want %v", string(util.MustJsonMarshal(got)), string(util.MustJsonMarshal(tt.want)))
			}
		})
	}
}
