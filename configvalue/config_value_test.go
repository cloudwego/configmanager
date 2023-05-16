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

package configvalue

import (
	"reflect"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/configmanager/configvalue/items"
	"github.com/cloudwego/configmanager/iface"
	"github.com/cloudwego/configmanager/util"
)

const (
	itemTypeInt64  iface.ItemType = "item-int64"
	itemTypePair   iface.ItemType = "item-pair"
	itemTypeBool   iface.ItemType = "item-bool"
	itemTypeString iface.ItemType = "item-string"
)

var itemPair = &items.ItemPair{
	Key:   "pair-key",
	Value: "pair-value",
}

func TestConfigValueImpl_GetItemOrDefault(t *testing.T) {
	type fields struct {
		items map[iface.ItemType]iface.ConfigValueItem
	}
	type args struct {
		itemType    iface.ItemType
		defaultItem iface.ConfigValueItem
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   iface.ConfigValueItem
	}{
		{
			name: "miss",
			fields: fields{
				items: nil,
			},
			args: args{
				itemType:    itemTypePair,
				defaultItem: items.CopyDefaultItemPair(),
			},
			want: items.CopyDefaultItemPair(),
		},
		{
			name: "hit",
			fields: fields{
				items: map[iface.ItemType]iface.ConfigValueItem{
					itemTypePair: itemPair,
				},
			},
			args: args{
				itemType:    itemTypePair,
				defaultItem: items.CopyDefaultItemPair(),
			},
			want: itemPair,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigValueImpl(tt.fields.items)
			if got := c.GetItemOrDefault(tt.args.itemType, tt.args.defaultItem); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetItemOrDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigValueImpl_SetItem(t *testing.T) {
	type fields struct {
		items map[iface.ItemType]iface.ConfigValueItem
	}
	type args struct {
		itemType iface.ItemType
		item     iface.ConfigValueItem
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ConfigValueImpl
	}{
		{
			name: "set-new",
			fields: fields{
				items: nil,
			},
			args: args{
				itemType: itemTypePair,
				item:     itemPair,
			},
			want: &ConfigValueImpl{
				items: map[iface.ItemType]iface.ConfigValueItem{
					itemTypePair: itemPair,
				},
			},
		},
		{
			name: "set-exist",
			fields: fields{
				items: map[iface.ItemType]iface.ConfigValueItem{
					itemTypePair: itemPair,
				},
			},
			args: args{
				itemType: itemTypePair,
				item:     items.CopyDefaultItemPair(),
			},
			want: &ConfigValueImpl{
				items: map[iface.ItemType]iface.ConfigValueItem{
					itemTypePair: items.CopyDefaultItemPair(),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigValueImpl(tt.fields.items)
			if got := c.SetItem(tt.args.itemType, tt.args.item); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SetItem() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigValueImpl_MarshalJSON(t *testing.T) {
	type fields struct {
		items map[iface.ItemType]iface.ConfigValueItem
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{
		{
			name: "empty",
			fields: fields{
				items: nil,
			},
			want:    []byte("{}"),
			wantErr: false,
		},
		{
			name: "normal",
			fields: fields{
				items: map[iface.ItemType]iface.ConfigValueItem{
					itemTypePair:   itemPair,
					itemTypeInt64:  items.ItemInt64Iface(items.Int64Max),
					itemTypeBool:   items.CopyItemBoolTrue(),
					itemTypeString: items.ItemStringIface("hello"),
				},
			},
			want:    []byte(`{"item-pair":{"key":"pair-key","value":"pair-value"}, "item-int64": 9223372036854775807, "item-bool": true, "item-string": "hello"}`),
			wantErr: false,
		},
	}
	jsonEqual := func(a, b []byte) bool {
		x := &map[string]interface{}{}
		y := &map[string]interface{}{}
		util.MustJsonUnmarshal(a, x)
		util.MustJsonUnmarshal(b, y)
		return reflect.DeepEqual(x, y)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConfigValueImpl(tt.fields.items)
			got, err := sonic.ConfigDefault.Marshal(c)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !jsonEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}
