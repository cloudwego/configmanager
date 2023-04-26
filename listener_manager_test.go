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

package configmanager

import (
	"reflect"
	"testing"

	"github.com/cloudwego/configmanager/iface"
)

func Test_newListenerManager(t *testing.T) {
	tests := []struct {
		name string
		want *listenerManager
	}{
		{
			name: "test new listener manager",
			want: &listenerManager{
				listeners: make(map[string]iface.ConfigChangeListener),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newListenerManager(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newListenerManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_listenerManager_registerListener(t *testing.T) {
	type args struct {
		uniqueID string
		listener iface.ConfigChangeListener
	}
	tests := []struct {
		name string
		args []args
		isOK func(m *listenerManager) bool
	}{
		{
			name: "one-listener",
			args: []args{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
			},
			isOK: func(m *listenerManager) bool {
				return m.size() == 1
			},
		},
		{
			name: "two-listener-same-id",
			args: []args{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
			},
			isOK: func(m *listenerManager) bool {
				return m.size() == 1
			},
		},
		{
			name: "two-listener-diff-id",
			args: []args{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
				{
					uniqueID: "service2",
					listener: func(change iface.ConfigChange) {},
				},
			},
			isOK: func(m *listenerManager) bool {
				return m.size() == 2
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newListenerManager()
			for _, arg := range tt.args {
				m.registerListener(arg.uniqueID, arg.listener)
			}
			if !tt.isOK(m) {
				t.Errorf("listenerManager.registerListener() = %v, want %v", m, tt.args)
			}
		})
	}
}

func Test_listenerManager_DeregisterListener(t *testing.T) {
	type registerArgs struct {
		uniqueID string
		listener iface.ConfigChangeListener
	}
	type deregisterArgs struct {
		uniqueID string
	}
	tests := []struct {
		name  string
		rargs []registerArgs
		dargs []deregisterArgs
		isOK  func(m *listenerManager) bool
	}{
		{
			name: "deregister-valid-unique-id",
			rargs: []registerArgs{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
			},
			dargs: []deregisterArgs{
				{
					uniqueID: "service1",
				},
			},
			isOK: func(m *listenerManager) bool {
				return m.size() == 0
			},
		},
		{
			name: "deregister-invalid-unique-id",
			rargs: []registerArgs{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
			},
			dargs: []deregisterArgs{
				{
					uniqueID: "service2",
				},
			},
			isOK: func(m *listenerManager) bool {
				return m.size() == 1
			},
		},
		{
			name: "deregister-valid-unique-id-twice",
			rargs: []registerArgs{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
			},
			dargs: []deregisterArgs{
				{
					uniqueID: "service1",
				},
				{
					uniqueID: "service1",
				},
			},
			isOK: func(m *listenerManager) bool {
				return m.size() == 0
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newListenerManager()
			for _, arg := range tt.rargs {
				m.registerListener(arg.uniqueID, arg.listener)
			}
			for _, arg := range tt.dargs {
				m.deregisterListener(arg.uniqueID)
			}
			if !tt.isOK(m) {
				t.Errorf("listenerManager.deregisterListener() err")
			}
		})
	}
}

func Test_listenerManager_copyServiceListeners(t *testing.T) {
	type args struct {
		uniqueID string
		listener iface.ConfigChangeListener
	}
	listener1 := func(change iface.ConfigChange) {}
	listener2 := func(change iface.ConfigChange) {}
	tests := []struct {
		name string
		args []args
		isOK func(listeners []iface.ConfigChangeListener) bool
	}{
		{
			name: "no listener",
			args: []args{},
			isOK: func(listeners []iface.ConfigChangeListener) bool {
				return len(listeners) == 0
			},
		},
		{
			name: "one listener",
			args: []args{
				{
					uniqueID: "service1",
					listener: listener1,
				},
			},
			isOK: func(listeners []iface.ConfigChangeListener) bool {
				return len(listeners) == 1 && reflect.ValueOf(listeners[0]).Pointer() == reflect.ValueOf(listener1).Pointer()
			},
		},
		{
			name: "two listener",
			args: []args{
				{
					uniqueID: "service1",
					listener: listener1,
				},
				{
					uniqueID: "service2",
					listener: listener2,
				},
			},
			isOK: func(listeners []iface.ConfigChangeListener) bool {
				return len(listeners) == 2 &&
					((reflect.ValueOf(listeners[0]).Pointer() == reflect.ValueOf(listener1).Pointer() && reflect.ValueOf(listeners[1]).Pointer() == reflect.ValueOf(listener2).Pointer()) ||
						(reflect.ValueOf(listeners[0]).Pointer() == reflect.ValueOf(listener2).Pointer() && reflect.ValueOf(listeners[1]).Pointer() == reflect.ValueOf(listener1).Pointer()))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newListenerManager()
			for _, arg := range tt.args {
				m.registerListener(arg.uniqueID, arg.listener)
			}
			if got := m.copyServiceListeners(); !tt.isOK(got) {
				t.Errorf("copyServiceListeners() = %v", got)
			}
		})
	}
}

func Test_listenerManager_size(t *testing.T) {
	type args struct {
		uniqueID string
		listener iface.ConfigChangeListener
	}
	tests := []struct {
		name string
		args []args
		want int
	}{
		{
			name: "no listener",
			args: []args{},
			want: 0,
		},
		{
			name: "one listener",
			args: []args{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
			},
			want: 1,
		},
		{
			name: "two listener",
			args: []args{
				{
					uniqueID: "service1",
					listener: func(change iface.ConfigChange) {},
				},
				{
					uniqueID: "service2",
					listener: func(change iface.ConfigChange) {},
				},
			},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newListenerManager()
			for _, arg := range tt.args {
				m.registerListener(arg.uniqueID, arg.listener)
			}
			if got := m.size(); got != tt.want {
				t.Errorf("size() = %v, want %v", got, tt.want)
			}
		})
	}
}
