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
	"reflect"
	"sync"
	"testing"

	"github.com/cloudwego/configmanager/configvalue/items"
	"github.com/cloudwego/configmanager/util"

	"github.com/cloudwego/configmanager/iface"
)

var _ iface.ConfigValueItem = &ItemInt{}

var (
	itemTypeInt      iface.ItemType = "item-int"
	itemTypePair     iface.ItemType = "item-pair"
	itemTypeAllowXXX iface.ItemType = "item-allow-xxx"
	itemTypeDenyXXX  iface.ItemType = "item-deny-xxx"
	itemTypeString   iface.ItemType = "item-string"
)

type ItemInt struct {
	Value int `json:"value"`
}

func (i ItemInt) DeepCopy() iface.ConfigValueItem {
	return &ItemInt{
		Value: i.Value,
	}
}

func (i ItemInt) EqualsTo(item iface.ConfigValueItem) bool {
	return i.Value == item.(*ItemInt).Value
}

var NewItemInt = util.JsonInitializer(func() iface.ConfigValueItem {
	return &ItemInt{}
})

func itemFactorySyncMapToMap(m *sync.Map) map[iface.ItemType]iface.ItemInitializer {
	result := make(map[iface.ItemType]iface.ItemInitializer)
	m.Range(func(key, value interface{}) bool {
		result[key.(iface.ItemType)] = value.(iface.ItemInitializer)
		return true
	})
	return result
}

func TestItemFactory_RegisterInitializer(t *testing.T) {
	type fields struct {
		initializers map[iface.ItemType]iface.ItemInitializer
	}
	type args struct {
		itemType    iface.ItemType
		initializer iface.ItemInitializer
	}
	tests := []struct {
		name   string
		fields fields
		args   []args
		isOK   func(f *ItemFactory) bool
	}{
		{
			name: "nil-initializer-nil-register",
			fields: fields{
				initializers: nil,
			},
			args: []args{},
			isOK: func(f *ItemFactory) bool {
				return len(itemFactorySyncMapToMap(f.initializer)) == 0
			},
		},
		{
			name: "one-initializer-nil-register",
			fields: fields{
				initializers: map[iface.ItemType]iface.ItemInitializer{
					itemTypeInt: NewItemInt,
				},
			},
			args: []args{},
			isOK: func(f *ItemFactory) bool {
				return len(itemFactorySyncMapToMap(f.initializer)) == 1
			},
		},
		{
			name: "one-initializer-one-register",
			fields: fields{
				initializers: map[iface.ItemType]iface.ItemInitializer{
					itemTypeInt: NewItemInt,
				},
			},
			args: []args{
				{
					itemType:    itemTypePair,
					initializer: items.NewItemPair,
				},
			},
			isOK: func(f *ItemFactory) bool {
				return len(itemFactorySyncMapToMap(f.initializer)) == 2
			},
		},
		{
			name: "one-initializer-one-same-register",
			fields: fields{
				initializers: map[iface.ItemType]iface.ItemInitializer{
					itemTypeInt: NewItemInt,
				},
			},
			args: []args{
				{
					itemType:    itemTypeInt,
					initializer: NewItemInt,
				},
			},
			isOK: func(f *ItemFactory) bool {
				return len(itemFactorySyncMapToMap(f.initializer)) == 1
			},
		},
		{
			name: "nil-initializer-two-same-register",
			fields: fields{
				initializers: nil,
			},
			args: []args{
				{
					itemType:    itemTypeInt,
					initializer: NewItemInt,
				},
				{
					itemType:    itemTypeInt,
					initializer: NewItemInt,
				},
			},
			isOK: func(f *ItemFactory) bool {
				return len(itemFactorySyncMapToMap(f.initializer)) == 1
			},
		},
		{
			name: "nil-initializer-two-different-register",
			fields: fields{
				initializers: nil,
			},
			args: []args{
				{
					itemType:    itemTypeInt,
					initializer: NewItemInt,
				},
				{
					itemType:    itemTypePair,
					initializer: items.NewItemPair,
				},
			},
			isOK: func(f *ItemFactory) bool {
				return len(itemFactorySyncMapToMap(f.initializer)) == 2
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewItemFactory(tt.fields.initializers)
			for _, arg := range tt.args {
				f.RegisterInitializer(arg.itemType, arg.initializer)
			}
			if !tt.isOK(f) {
				t.Errorf("ItemFactory.RegisterInitializer() is not ok")
			}
		})
	}
}

func TestItemFactory_NewItem(t *testing.T) {
	type fields struct {
		initializer map[iface.ItemType]iface.ItemInitializer
	}
	type args struct {
		itemType iface.ItemType
		js       []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    iface.ConfigValueItem
		wantErr bool
	}{
		{
			name: "nil-initializer",
			fields: fields{
				initializer: nil,
			},
			args: args{
				itemType: itemTypeInt,
				js:       nil,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "rpc-timeout-initializer",
			fields: fields{
				initializer: map[iface.ItemType]iface.ItemInitializer{
					itemTypeInt: NewItemInt,
				},
			},
			args: args{
				itemType: itemTypeInt,
				js:       []byte(`{"value": 1}`),
			},
			want:    &ItemInt{Value: 1},
			wantErr: false,
		},
		{
			name: "retry-initializer-new-item-pair-no-initializer",
			fields: fields{
				initializer: map[iface.ItemType]iface.ItemInitializer{
					itemTypeInt: NewItemInt,
				},
			},
			args: args{
				itemType: itemTypePair,
				js:       []byte(`{"key": "k", "value": "v""}`),
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "retry-initializer-new-item-pair-ok",
			fields: fields{
				initializer: map[iface.ItemType]iface.ItemInitializer{
					itemTypePair: items.NewItemPair,
				},
			},
			args: args{
				itemType: itemTypePair,
				js:       []byte(`{"key": "k", "value": "v"}`),
			},
			want: &items.ItemPair{
				Key:   "k",
				Value: "v",
			},
			wantErr: false,
		},
		{
			name: "retry-initializer-new-item-pair-unmarshal-err",
			fields: fields{
				initializer: map[iface.ItemType]iface.ItemInitializer{
					itemTypePair: items.NewItemPair,
				},
			},
			args: args{
				itemType: itemTypePair,
				js:       []byte(`{"key": "k", "value": "v""`),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewItemFactory(tt.fields.initializer)
			if got, err := f.NewItem(tt.args.itemType, tt.args.js); (err != nil) != tt.wantErr || !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewItem() = %v, want %v", got, tt.want)
			}
		})
	}
}
