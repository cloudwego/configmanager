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
	"errors"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cloudwego/configmanager/configvalue/items"

	"github.com/cloudwego/configmanager/configvalue"
	"github.com/cloudwego/configmanager/iface"
	"github.com/cloudwego/configmanager/util"
)

type ConfigKeyUT struct {
	service string
	method  string
}

func (k *ConfigKeyUT) ToString() string {
	return k.service + "|" + k.method
}

const itemTypeTest iface.ItemType = "item-test"

var (
	defaultConfig = configvalue.NewConfigValueImpl(map[iface.ItemType]iface.ConfigValueItem{
		itemTypeTest: items.CopyDefaultItemPair(),
	})

	defaultCurrentConfig = map[string]iface.ConfigValue{
		"service1|method1": defaultConfig.DeepCopy(),
		"service1|method2": defaultConfig.DeepCopy(),
		"service2|method3": defaultConfig.DeepCopy(),
		"service2|method4": defaultConfig.DeepCopy(),
	}

	existKey = &ConfigKeyUT{
		service: "service1",
		method:  "method1",
	}

	notExistKey = &ConfigKeyUT{
		service: "service3",
		method:  "method5",
	}

	sampleConfig = func() iface.ConfigValue {
		value := defaultConfig.DeepCopy()
		item, err := value.GetItem(itemTypeTest)
		if err != nil {
			panic(err)
		}
		pair := item.(*items.ItemPair)
		pair.Key = "key"
		pair.Value = "value"
		return value
	}()
)

type ConfigProviderUT struct {
	// used to simulate config change
	nextConfig atomic.Value // map[string]iface.ConfigValue
	loadError  error
}

func NewConfigProviderUT(nextConfig map[string]iface.ConfigValue) *ConfigProviderUT {
	p := &ConfigProviderUT{}
	p.nextConfig.Store(nextConfig)
	return p
}

func (p *ConfigProviderUT) LoadConfig(currentConfig map[string]iface.ConfigValue) (map[string]iface.ConfigValue, error) {
	if p.loadError != nil {
		return nil, p.loadError
	}
	result := make(map[string]iface.ConfigValue)

	updater := func(c map[string]iface.ConfigValue) {
		for k, v := range c {
			result[k] = v.DeepCopy()
		}
	}

	if currentConfig != nil {
		updater(currentConfig)
	}
	if nextConfig, ok := p.nextConfig.Load().(map[string]iface.ConfigValue); ok {
		updater(nextConfig)
	}
	return result, nil
}

func TestConfigManager_GetConfig(t *testing.T) {
	type fields struct {
		currentConfig *sync.Map
	}
	type args struct {
		key iface.ConfigKey
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    iface.ConfigValue
		wantErr error
	}{
		{
			name: "key_not_exist",
			fields: fields{
				currentConfig: &sync.Map{},
			},
			args: args{
				key: notExistKey,
			},
			want:    nil,
			wantErr: iface.ErrConfigNotFound,
		},
		{
			name: "key_exist",
			fields: fields{
				currentConfig: func() *sync.Map {
					m := &sync.Map{}
					m.Store(existKey.ToString(), defaultCurrentConfig[existKey.ToString()])
					return m
				}(),
			},
			args: args{
				key: existKey,
			},
			want:    defaultCurrentConfig[existKey.ToString()],
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				currentConfig: tt.fields.currentConfig,
			}
			got, err := m.GetConfig(tt.args.key)
			if err != tt.wantErr {
				t.Errorf("GetConfig() error = %v, wantErr %v, key = %v", err, tt.wantErr, tt.args.key)
				return
			}
			if got == nil || tt.want == nil {
				if got != tt.want {
					t.Errorf("GetConfig() got = %v, want %v", got, tt.want)
				}
				return
			}
			if !got.EqualsTo(tt.want) {
				t.Errorf("GetConfig() got = %v, want %v", got, tt.want)
				return
			}
		})
	}

	t.Run("test_panic", func(t *testing.T) {
		m := &ConfigManager{
			currentConfig: func() *sync.Map {
				m := &sync.Map{}
				m.Store(existKey.ToString(), "Value")
				return m
			}(),
		}

		defer func() {
			if err := recover(); err == nil {
				t.Errorf("GetConfig() should panic")
			}
		}()
		val, err := m.GetConfig(existKey)
		t.Errorf("GetConfig() result: val = %v, err = %v", val, err)
	})
}

func TestConfigManager_callConfigChangeListeners(t *testing.T) {
	type fields struct {
		listeners *listenerManager
	}
	type args struct {
		differences []iface.ConfigChange
	}
	t1count := int32(0)
	t2count := int32(0)
	tests := []struct {
		name   string
		fields fields
		args   args
		result *int32
		want   int32
	}{
		{
			name: "no_listener",
			fields: fields{
				listeners: newListenerManager(),
			},
			args: args{
				differences: []iface.ConfigChange{
					{
						Type:      iface.ConfigChangeTypeUpdate,
						ConfigKey: "service1|method1",
						OldValue:  defaultConfig.DeepCopy(),
						NewValue:  sampleConfig,
					},
				},
			},
			want:   0,
			result: &t1count,
		},
		{
			name: "1_listener",
			fields: fields{
				listeners: func() *listenerManager {
					m := newListenerManager()
					m.registerListener("service1-listener", func(change iface.ConfigChange) {
						atomic.AddInt32(&t1count, 1)
					})
					return m
				}(),
			},
			args: args{
				differences: []iface.ConfigChange{
					{
						Type:      iface.ConfigChangeTypeUpdate,
						ConfigKey: "service1|method",
						OldValue:  defaultConfig.DeepCopy(),
						NewValue:  sampleConfig,
					},
				},
			},
			want:   1,
			result: &t1count,
		},
		{
			name: "2_listeners",
			fields: fields{
				listeners: func() *listenerManager {
					m := newListenerManager()
					m.registerListener("service2-listener1", func(change iface.ConfigChange) {
						atomic.AddInt32(&t2count, 1)
					})
					m.registerListener("service2-listener2", func(change iface.ConfigChange) {
						atomic.AddInt32(&t2count, 1)
					})
					return m
				}(),
			},
			args: args{
				differences: []iface.ConfigChange{
					{
						Type:      iface.ConfigChangeTypeUpdate,
						ConfigKey: "service2|method1",
						OldValue:  defaultConfig.DeepCopy(),
						NewValue:  sampleConfig,
					},
					{
						Type:      iface.ConfigChangeTypeUpdate,
						ConfigKey: "service2|method2",
						OldValue:  defaultConfig.DeepCopy(),
						NewValue:  sampleConfig,
					},
				},
			},
			want:   4,
			result: &t2count,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				listeners: tt.fields.listeners,
				logger:    log.Printf,
			}
			m.callConfigChangeListeners(tt.args.differences)
			if tt.want != *tt.result {
				t.Errorf("callConfigChangeListeners() want %v, result: %v", tt.want, *tt.result)
			}
		})
	}
}

func TestConfigManager_updateCurrentConfig(t *testing.T) {
	type fields struct {
		currentConfig *sync.Map
	}
	type args struct {
		latestConfig map[string]iface.ConfigValue
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "nil_to_non_nil",
			fields: fields{
				currentConfig: nil,
			},
			args: args{
				latestConfig: map[string]iface.ConfigValue{
					"service1|method1": defaultConfig.DeepCopy(),
				},
			},
		},
		{
			name: "non_nil_to_non_nil",
			fields: fields{
				currentConfig: func() *sync.Map {
					m := new(sync.Map)
					m.Store("service1|key1", sampleConfig)
					return m
				}(),
			},
			args: args{
				latestConfig: map[string]iface.ConfigValue{
					"service1|method1": defaultConfig.DeepCopy(),
					"service1|method2": sampleConfig,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				currentConfig: tt.fields.currentConfig,
			}
			m.updateCurrentConfig(tt.args.latestConfig)
			if !reflect.DeepEqual(tt.args.latestConfig, util.ConfigSyncMapToMap(m.currentConfig)) {
				t.Errorf("updateCurrentConfig() currentConfig = %v, want %v", m.currentConfig, tt.args.latestConfig)
			}
		})
	}
}

func TestConfigManager_shallowCopyConfig(t *testing.T) {
	type fields struct {
		currentConfig *sync.Map
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]iface.ConfigValue
	}{
		{
			name: "nil",
			fields: fields{
				currentConfig: nil,
			},
			want: make(map[string]iface.ConfigValue), // always get a non-nil map
		},
		{
			name: "non_nil",
			fields: fields{
				currentConfig: func() *sync.Map {
					m := new(sync.Map)
					m.Store("service1|method1", defaultConfig.DeepCopy())
					return m
				}(),
			},
			want: map[string]iface.ConfigValue{
				"service1|method1": defaultConfig.DeepCopy(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				currentConfig: tt.fields.currentConfig,
			}
			if got := m.shallowCopyConfig(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shallowCopyConfig() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigManager_Stop(t *testing.T) {
	t.Run("stop", func(t *testing.T) {
		m := &ConfigManager{
			manualRefreshSignal: make(chan iface.RefreshCompleteSignal, 100),
			stopSignal:          make(chan struct{}, 1),
		}
		m.Stop()
		_, stopped := m.waitForSignal()
		if !stopped {
			t.Errorf("stop failed")
		}
	})
}

func TestConfigManager_RegisterConfigChangeListener(t *testing.T) {
	type fields struct {
		listeners *listenerManager
	}
	type args struct {
		listenerID string
		listener   iface.ConfigChangeListener
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "1_listener",
			fields: fields{
				listeners: newListenerManager(),
			},
			args: args{
				listenerID: "1_listener",
				listener:   func(change iface.ConfigChange) {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				listeners: tt.fields.listeners,
			}
			m.RegisterConfigChangeListener(tt.args.listenerID, tt.args.listener)
			got := m.listeners.copyServiceListeners()
			if len(got) != 1 {
				t.Errorf("RegisterConfigChangeListener() got = %v, want %v", got, 1)
				return
			}
			if reflect.ValueOf(got[0]).Pointer() != reflect.ValueOf(tt.args.listener).Pointer() {
				t.Errorf("RegisterConfigChangeListener() got = %v, want %v", got, tt.args.listener)
				return
			}
			t.Logf("RegisterConfigChangeListener() got = %v", got)
		})
	}
}

func TestConfigManager_sendRefreshSignal(t *testing.T) {
	type fields struct {
		refreshSignal chan iface.RefreshCompleteSignal
	}
	type args struct {
		count int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "1_signal",
			fields: fields{
				refreshSignal: make(chan iface.RefreshCompleteSignal, 10),
			},
			args: args{
				count: 1,
			},
		},
		{
			name: "3_signal",
			fields: fields{
				refreshSignal: make(chan iface.RefreshCompleteSignal, 10),
			},
			args: args{
				count: 3, // shouldn't block
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				manualRefreshSignal: tt.fields.refreshSignal,
			}
			for i := 0; i < tt.args.count; i++ {
				m.sendRefreshSignal()
			}
			waiters := collectWaitersNonBlocking(m.manualRefreshSignal)
			if len(waiters) != tt.args.count {
				t.Errorf("sendRefreshSignal() waiters = %v, want %v", len(waiters), tt.args.count)
				return
			}
		})
	}

	t.Run("wait-for-refresh", func(t *testing.T) {
		m := NewConfigManager([]ConfigManagerOption{
			WithConfigProvider(NewConfigProviderUT(defaultCurrentConfig)),
			WithErrorLogger(log.Printf),
			WithRefreshInterval(0),
		})
		done := m.sendRefreshSignal()
		select {
		case <-done:
			if len(m.shallowCopyConfig()) != len(defaultCurrentConfig) {
				t.Errorf("sendRefreshSignal() config = %v, want %v", m.shallowCopyConfig(), defaultCurrentConfig)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("sendRefreshSignal() timeout")
		}
	})

	t.Run("wait-for-refresh-fail", func(t *testing.T) {
		p := NewConfigProviderUT(defaultCurrentConfig)
		m := NewConfigManager([]ConfigManagerOption{
			WithConfigProvider(p),
			WithErrorLogger(log.Printf),
			WithRefreshInterval(0),
		})
		p.loadError = errors.New("load error")
		done := m.sendRefreshSignal()
		select {
		case err := <-done:
			if len(m.shallowCopyConfig()) != 0 {
				t.Errorf("load err, shouldn't update config")
			}
			if err != p.loadError {
				t.Errorf("sendRefreshSignal() err = %v, want %v", err, p.loadError)
			}
		case <-time.After(1 * time.Second):
			t.Errorf("sendRefreshSignal() timeout")
		}
	})

	t.Run("multiple waiter", func(t *testing.T) {
		p := NewConfigProviderUT(defaultCurrentConfig)
		m := NewConfigManager([]ConfigManagerOption{
			WithConfigProvider(p),
			WithErrorLogger(log.Printf),
			WithRefreshInterval(0),
		})
		done1 := m.sendRefreshSignal()
		done2 := m.sendRefreshSignal()
		done3 := m.sendRefreshSignal()
		for _, done := range []iface.RefreshCompleteSignal{done1, done2, done3} {
			select {
			case <-done:
			case <-time.After(1 * time.Second):
				t.Errorf("sendRefreshSignal() timeout")
			}
		}
	})
}

func TestConfigManager_validate(t *testing.T) {
	type fields struct {
		configProvider iface.ConfigProvider
	}
	tests := []struct {
		name      string
		fields    fields
		wantPanic bool
	}{
		{
			name: "nil_config_provider",
			fields: fields{
				configProvider: nil,
			},
			wantPanic: true,
		},
		{
			name: "nil_config_key_parser",
			fields: fields{
				configProvider: NewConfigProviderUT(defaultCurrentConfig),
			},
			wantPanic: true,
		},
		{
			name: "no_panic",
			fields: fields{
				configProvider: NewConfigProviderUT(defaultCurrentConfig),
			},
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				configProvider: tt.fields.configProvider,
			}

			w := sync.WaitGroup{}
			w.Add(1)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						if !tt.wantPanic {
							t.Errorf("validate() panic = %v, wantPanic %v", r, tt.wantPanic)
						}
					}
					w.Done()
				}()
				m.validate()
			}()
			w.Wait()
		})
	}
}

func TestWithConfigSerializer(t *testing.T) {
	type args struct {
		serializer iface.ConfigSerializer
	}
	tests := []struct {
		name string
		args args
		want iface.ConfigSerializer
	}{
		{
			name: "jsonSerializer",
			args: args{
				serializer: util.JSONSerializer,
			},
			want: util.JSONSerializer,
		},
		{
			name: "nil",
			args: args{
				serializer: nil,
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{}
			WithConfigSerializer(tt.args.serializer)(m)
			if reflect.ValueOf(m.configSerializer) != reflect.ValueOf(tt.want) {
				t.Errorf("WithConfigSerializer() = %v, want %v", m.configSerializer, tt.want)
			}
		})
	}
}

func TestWithLogLimiter(t *testing.T) {
	t.Run("no-limiter", func(t *testing.T) {
		counter := int32(0)
		m := NewConfigManager([]ConfigManagerOption{
			WithConfigProvider(NewConfigProviderUT(defaultCurrentConfig)),
			WithErrorLogger(func(format string, v ...interface{}) {
				atomic.AddInt32(&counter, 1)
			}),
		})
		m.log("test")
		m.log("test")
		if atomic.LoadInt32(&counter) != 2 {
			t.Errorf("log() should not be limited")
		}
	})
	t.Run("with-limiter", func(t *testing.T) {
		counter := int32(0)
		limiter := util.NewFrequencyLimiter(time.Second, 1)
		defer limiter.Stop()
		m := NewConfigManager([]ConfigManagerOption{
			WithConfigProvider(NewConfigProviderUT(defaultCurrentConfig)),
			WithErrorLogger(func(format string, v ...interface{}) {
				atomic.AddInt32(&counter, 1)
			}),
			WithLogLimiter(limiter),
		})
		m.log("test")
		m.log("test")
		if atomic.LoadInt32(&counter) != 1 {
			t.Errorf("log() should be limited")
		}
	})
}

func TestWithErrorLogger(t *testing.T) {
	type args struct {
		logger iface.LogFunc
	}
	tests := []struct {
		name string
		args args
		want iface.LogFunc
	}{
		{
			name: "nil_logger",
			args: args{
				logger: nil,
			},
			want: nil,
		},
		{
			name: "not_nil_logger",
			args: args{
				logger: log.Printf,
			},
			want: log.Printf,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{}
			WithErrorLogger(tt.args.logger)(m)
			if reflect.ValueOf(m.logger).Pointer() != reflect.ValueOf(tt.want).Pointer() {
				t.Errorf("WithErrorLogger() = %v, want %v", m.logger, tt.want)
			}
		})
	}
}

func TestWithConfigProvider(t *testing.T) {
	type args struct {
		provider iface.ConfigProvider
	}
	provider := NewConfigProviderUT(defaultCurrentConfig)

	tests := []struct {
		name string
		args args
		want iface.ConfigProvider
	}{
		{
			name: "nil_provider",
			args: args{
				provider: nil,
			},
			want: nil,
		},
		{
			name: "not_nil_provider",
			args: args{
				provider: provider,
			},
			want: provider,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{}
			WithConfigProvider(tt.args.provider)(m)
			if m.configProvider != tt.want {
				t.Errorf("WithConfigProvider() = %v, want %v", m.configProvider, tt.want)
			}
		})
	}
}

func TestWithRefreshInterval(t *testing.T) {
	type args struct {
		interval time.Duration
	}
	tests := []struct {
		name string
		args args
		want time.Duration
	}{
		{
			name: "zero_interval",
			args: args{
				interval: 0,
			},
			want: 0,
		},
		{
			name: "not_zero_interval",
			args: args{
				interval: 1 * time.Second,
			},
			want: 1 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{}
			WithRefreshInterval(tt.args.interval)(m)
			if m.refreshInterval != tt.want {
				t.Errorf("WithRefreshInterval() = %v, want %v", m.refreshInterval, tt.want)
			}
		})
	}
}

func TestNewConfigManager(t *testing.T) {
	t.Run("config manager", func(t *testing.T) {
		p := NewConfigProviderUT(defaultCurrentConfig)
		m := NewConfigManager([]ConfigManagerOption{
			WithConfigProvider(p),
			WithErrorLogger(log.Printf),
			WithRefreshInterval(1 * time.Second),
		})
		if m == nil {
			t.Errorf("NewConfigManager() = %v, want not nil", m)
			return
		}
		if err := m.RefreshAndWait(); err != nil {
			t.Errorf("RefreshAndWait() error = %v, wantErr %v", err, nil)
			return
		}
		if len(m.shallowCopyConfig()) != len(defaultCurrentConfig) {
			t.Errorf("NewConfigManager() = %v, want 4 config", m.shallowCopyConfig())
			return
		}

		nextConfig := map[string]iface.ConfigValue{
			"service3|method1": defaultConfig.DeepCopy(),
		}
		p.nextConfig.Store(nextConfig)

		if err := m.RefreshAndWait(); err != nil {
			t.Errorf("RefreshAndWait() error = %v, wantErr %v", err, nil)
			return
		}
		if len(m.shallowCopyConfig()) != len(defaultCurrentConfig)+1 {
			t.Errorf("NewConfigManager() = %v, want 5 config", m.shallowCopyConfig())
			return
		}
		t.Logf("loaded config: %v", m.shallowCopyConfig())
	})
}

func TestConfigManager_Dump(t *testing.T) {
	type fields struct {
		currentConfig    *sync.Map
		configSerializer iface.ConfigSerializer
	}
	type args struct {
		filepath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    string
	}{
		{
			name: "dump_success",
			fields: fields{
				currentConfig:    &sync.Map{},
				configSerializer: util.JSONSerializer,
			},
			args: args{
				filepath: "dump_test1.json",
			},
			wantErr: false,
			want:    "{}",
		},
		{
			name: "dump_success",
			fields: fields{
				currentConfig: func() *sync.Map {
					m := &sync.Map{}
					m.Store("key", defaultConfig)
					return m
				}(),
				configSerializer: util.JSONSerializer,
			},
			args: args{
				filepath: "dump_test2.json",
			},
			wantErr: false,
			want: func() string {
				s, _ := util.JSONSerializer.Encode(map[string]iface.ConfigValue{"key": defaultConfig})
				return string(s)
			}(),
		},
		{
			name: "dump_fail",
			fields: fields{
				currentConfig:    &sync.Map{},
				configSerializer: nil,
			},
			args: args{
				filepath: "dump_test3.json",
			},
			wantErr: true,
			want:    "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				currentConfig:    tt.fields.currentConfig,
				configSerializer: tt.fields.configSerializer,
			}

			defer func() {
				_ = os.Remove(tt.args.filepath)
			}()

			if err := m.Dump(tt.args.filepath); (err != nil) != tt.wantErr {
				t.Errorf("Dump() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				content, err := ioutil.ReadFile(tt.args.filepath)
				if err != nil {
					t.Errorf("Dump() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if string(content) != tt.want {
					t.Errorf("Dump() content = %v, want %v", string(content), tt.want)
					return
				}
			}
		})
	}
}

func TestConfigManager_DeregisterConfigChangeListener(t *testing.T) {
	type fields struct {
		listeners *listenerManager
	}
	type args struct {
		identifier string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "deregister_same_id",
			fields: fields{
				listeners: func() *listenerManager {
					lm := newListenerManager()
					lm.registerListener("id1", func(c iface.ConfigChange) {})
					return lm
				}(),
			},
			args: args{
				identifier: "id1",
			},
			want: 0,
		},
		{
			name: "deregister_diff_id",
			fields: fields{
				listeners: func() *listenerManager {
					lm := newListenerManager()
					lm.registerListener("id1", func(c iface.ConfigChange) {})
					return lm
				}(),
			},
			args: args{
				identifier: "id2",
			},
			want: 1,
		},
		{
			name: "deregister_from_empty",
			fields: fields{
				listeners: newListenerManager(),
			},
			args: args{
				identifier: "id1",
			},
			want: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				listeners: tt.fields.listeners,
			}
			m.DeregisterConfigChangeListener(tt.args.identifier)
			if tt.fields.listeners.size() != tt.want {
				t.Errorf("DeregisterConfigChangeListener() = %v, want %v", tt.fields.listeners.size(), tt.want)
			}
		})
	}
}

func TestConfigManager_GetConfigItem(t *testing.T) {
	type fields struct {
		currentConfig *sync.Map
	}
	type args struct {
		key      iface.ConfigKey
		itemType iface.ItemType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    iface.ConfigValueItem
		wantErr bool
	}{
		{
			name: "get_from_empty",
			fields: fields{
				currentConfig: &sync.Map{},
			},
			args: args{
				key: &ConfigKeyUT{
					service: "service1",
					method:  "method1",
				},
				itemType: itemTypeTest,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				currentConfig: tt.fields.currentConfig,
			}
			got, err := m.GetConfigItem(tt.args.key, tt.args.itemType)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfigItem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConfigItem() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigManager_GetProvider(t *testing.T) {
	type fields struct {
		configProvider iface.ConfigProvider
	}
	providerUT := NewConfigProviderUT(nil)
	tests := []struct {
		name   string
		fields fields
		want   iface.ConfigProvider
	}{
		{
			name: "nil provider",
			fields: fields{
				configProvider: nil,
			},
			want: nil,
		},
		{
			name: "not nil provider",
			fields: fields{
				configProvider: providerUT,
			},
			want: providerUT,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &ConfigManager{
				configProvider: tt.fields.configProvider,
			}
			if got := m.GetProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigManager_waitTimer(t *testing.T) {
	t.Run("no timeout", func(t *testing.T) {
		m := &ConfigManager{
			waitTimeout: 0,
		}
		select {
		case <-m.waitTimer():
			t.Errorf("waitTimer() should not be triggered")
		default:
		}
	})

	t.Run("timeout", func(t *testing.T) {
		m := &ConfigManager{
			waitTimeout: time.Millisecond,
		}
		select {
		case <-m.waitTimer():
		case <-time.After(time.Second):
			t.Errorf("waitTimer() should be triggered")
		}
	})
}

func TestConfigManager_waitForSignal(t *testing.T) {
	t.Run("no refreshInterval", func(t *testing.T) {
		m := &ConfigManager{
			manualRefreshSignal: make(chan iface.RefreshCompleteSignal, 100),
			tickerRefreshSignal: make(<-chan time.Time),
		}
		m.Refresh()
		m.Refresh()
		waiters, _ := m.waitForSignal()
		if len(waiters) != 2 {
			t.Errorf("waitForSignal() = %v, want %v", len(waiters), 2)
		}
	})

	t.Run("no waiter", func(t *testing.T) {
		m := &ConfigManager{
			manualRefreshSignal: make(chan iface.RefreshCompleteSignal, 100),
			tickerRefreshSignal: time.After(time.Millisecond),
		}
		time.Sleep(time.Millisecond * 2)
		waiters, _ := m.waitForSignal()
		if len(waiters) != 0 {
			t.Errorf("waitForSignal() = %v, want %v", len(waiters), 0)
		}
	})
}

func TestConfigManager_initRefreshTicker(t *testing.T) {
	t.Run("no refresh interval", func(t *testing.T) {
		m := &ConfigManager{
			refreshInterval: 0,
		}
		m.initRefreshTicker()
		if m.refreshTicker != nil {
			t.Errorf("initRefreshTicker(): refreshTicker is not nil")
		}
		select {
		case <-m.tickerRefreshSignal:
			t.Errorf("initRefreshTicker(): tickerRefreshSignal should not be triggered")
		default:
		}
	})

	t.Run("with refresh interval", func(t *testing.T) {
		m := &ConfigManager{
			refreshInterval: time.Millisecond,
		}
		m.initRefreshTicker()
		if m.refreshTicker == nil {
			t.Errorf("initRefreshTicker(): refreshTicker is nil")
		}
		time.Sleep(time.Millisecond * 2)
		select {
		case <-m.tickerRefreshSignal:
		default:
			t.Errorf("initRefreshTicker(): tickerRefreshSignal should be triggered")
		}
	})
}

func TestConfigManager_loadAndUpdateConfig(t *testing.T) {
	t.Run("load err", func(t *testing.T) {
		err := errors.New("load err")
		p := NewConfigProviderUT(nil)
		m := &ConfigManager{
			currentConfig:  &sync.Map{},
			configProvider: p,
			listeners:      newListenerManager(),
			logger:         log.Printf,
		}
		p.loadError = err
		if err1 := m.loadAndUpdateConfig(); err != err1 {
			t.Errorf("loadAndUpdateConfig() err = %v, wantErr %v", err1, err)
		}
	})

	t.Run("not modified", func(t *testing.T) {
		p := NewConfigProviderUT(defaultCurrentConfig)
		count := 0
		m := &ConfigManager{
			currentConfig:  &sync.Map{},
			configProvider: p,
			listeners: func() *listenerManager {
				m := newListenerManager()
				m.registerListener("key1", func(change iface.ConfigChange) {
					count += 1
				})
				return m
			}(),
			logger: log.Printf,
		}

		p.loadError = iface.ErrNotModified
		if err := m.loadAndUpdateConfig(); err != nil {
			t.Errorf("loadAndUpdateConfig() err = %v, wantErr nil", err)
			return
		}

		// will not update currentConfig
		if len(m.shallowCopyConfig()) != 0 {
			t.Errorf("loadAndUpdateConfig() should not update currentConfig")
			return
		}

		// won't call listener
		if count != 0 {
			t.Errorf("loadAndUpdateConfig() should not trigger listener")
			return
		}
	})

	t.Run("normal refresh", func(t *testing.T) {
		p := NewConfigProviderUT(defaultCurrentConfig)
		count := 0
		m := &ConfigManager{
			currentConfig:  &sync.Map{},
			configProvider: p,
			listeners: func() *listenerManager {
				m := newListenerManager()
				m.registerListener("key1", func(change iface.ConfigChange) {
					count += 1
				})
				return m
			}(),
			logger: log.Printf,
		}

		if err := m.loadAndUpdateConfig(); err != nil {
			t.Errorf("loadAndUpdateConfig() err = %v, wantErr nil", err)
			return
		}

		// will not update currentConfig
		if len(m.shallowCopyConfig()) == 0 {
			t.Errorf("loadAndUpdateConfig() should update currentConfig")
			return
		}

		// won't call listener
		if count == 0 {
			t.Errorf("loadAndUpdateConfig() should trigger listener")
			return
		}
	})
}

func TestWithWaitTimeout(t *testing.T) {
	m := &ConfigManager{}
	WithWaitTimeout(time.Millisecond)(m)
	if m.waitTimeout != time.Millisecond {
		t.Errorf("WithWaitTimeout() should set waitTimeout")
	}
}
