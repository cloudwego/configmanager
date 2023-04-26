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
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/cloudwego/configmanager/iface"
	"github.com/cloudwego/configmanager/util"
)

const (
	defaultRefreshInterval = 10 * time.Second
	defaultWaitTimeout     = 5 * time.Second
)

var (
	_ iface.ConfigManagerIface = &ConfigManager{} // ensure ConfigManager implements ConfigManagerIface

	// emptyTimeChannel is a channel that never receives any value.
	emptyTimeChannel = make(<-chan time.Time)
)

// ConfigManagerOption is a function type that modifies the configuration of a ConfigManager
type ConfigManagerOption func(*ConfigManager)

// ConfigManager is a struct that manages configuration data, including refreshing it at regular intervals.
// It uses a read-only sync.Map to store the current configuration, and provides methods for accessing it.
type ConfigManager struct {
	refreshInterval time.Duration
	currentConfig   *sync.Map // map[iface.ConfigKey.ToString()]iface.ConfigValue, should always be read-only after loaded

	configProvider   iface.ConfigProvider
	configSerializer iface.ConfigSerializer

	listeners *listenerManager

	logger     iface.LogFunc
	logLimiter iface.Limiter

	waitTimeout time.Duration // 0 for no timeout

	// manualRefreshSignal is trigger by calling Refresh() or RefreshAndWait()
	manualRefreshSignal chan iface.RefreshCompleteSignal
	// tickerRefreshSignal is triggered by the ticker if refreshInterval > 0
	refreshTicker       *time.Ticker
	tickerRefreshSignal <-chan time.Time

	stopSignal chan struct{}
}

// GetAllConfig retrieves all configuration values stored in the ConfigManager's map
// and returns it in a map with a string key and iface.ConfigValue value.
// **DO NOT MODIFY** values in the returned map.
func (m *ConfigManager) GetAllConfig() map[string]iface.ConfigValue {
	return m.shallowCopyConfig()
}

// NewConfigManager creates a new instance of ConfigManager with the given options.
func NewConfigManager(options []ConfigManagerOption) *ConfigManager {
	m := &ConfigManager{
		refreshInterval:     defaultRefreshInterval,
		waitTimeout:         defaultWaitTimeout,
		currentConfig:       &sync.Map{},
		configProvider:      nil,
		configSerializer:    util.JSONSerializer,
		listeners:           newListenerManager(),
		logger:              log.Printf,
		manualRefreshSignal: make(chan iface.RefreshCompleteSignal, 100),
		stopSignal:          make(chan struct{}, 1),
	}

	for _, option := range options {
		option(m)
	}

	m.validate()

	go m.refreshConfigLoop()
	return m
}

// WithRefreshInterval sets the interval time for refreshing the configuration.
// If not set, the interval defaults to be defaultRefreshInterval
func WithRefreshInterval(interval time.Duration) ConfigManagerOption {
	return func(m *ConfigManager) {
		m.refreshInterval = interval
	}
}

// WithConfigProvider sets the given ConfigProvider as the source for configuration data.
func WithConfigProvider(provider iface.ConfigProvider) ConfigManagerOption {
	return func(m *ConfigManager) {
		m.configProvider = provider
	}
}

// WithErrorLogger sets the error logger for the ConfigManager instance
// If not set, the logger defaults to be log.Printf
func WithErrorLogger(logger iface.LogFunc) ConfigManagerOption {
	return func(m *ConfigManager) {
		m.logger = logger
	}
}

// WithConfigSerializer sets the ConfigSerializer for the ConfigManager.
// If not set, the serializer defaults to be util.JSONSerializer
func WithConfigSerializer(serializer iface.ConfigSerializer) ConfigManagerOption {
	return func(m *ConfigManager) {
		m.configSerializer = serializer
	}
}

// WithLogLimiter sets a Limiter for logging in ConfigManager.
func WithLogLimiter(limiter iface.Limiter) ConfigManagerOption {
	return func(m *ConfigManager) {
		m.logLimiter = limiter
	}
}

// WithWaitTimeout sets the timeout for waiting for the configuration update to complete.
// if not set, the timeout defaults to be defaultWaitTimeout
func WithWaitTimeout(timeout time.Duration) ConfigManagerOption {
	return func(m *ConfigManager) {
		m.waitTimeout = timeout
	}
}

// validate checks necessary components; fail fast if not valid
func (m *ConfigManager) validate() {
	if m.configProvider == nil {
		panic("ConfigManager.validate(): config provider is nil")
	}
}

// log logs the given message when the logger is not nil and the logLimiter allows it.
func (m *ConfigManager) log(format string, args ...interface{}) {
	if m.logger == nil {
		return
	}
	if m.logLimiter == nil || m.logLimiter.Allow() {
		m.logger(format, args...)
	}
}

// GetProvider returns the ConfigProvider interface associated with the ConfigManager instance.
// In some scenarios, there's need to call the provider's API
func (m *ConfigManager) GetProvider() iface.ConfigProvider {
	return m.configProvider
}

// sendRefreshSignal sends a refresh signal to m.manualRefreshSignal to update its configuration.
// An iface.RefreshCompleteSignal is returned if the caller needs to wait for the refresh to complete.
func (m *ConfigManager) sendRefreshSignal() iface.RefreshCompleteSignal {
	waiter := make(iface.RefreshCompleteSignal, 1) // buffered channel to avoid blocking
	m.manualRefreshSignal <- waiter
	return waiter
}

// Refresh is called to refresh the configuration manager by sending a refresh signal.
// It may block if there are lots of callers
func (m *ConfigManager) Refresh() iface.RefreshCompleteSignal {
	return m.sendRefreshSignal()
}

// RefreshAndWait updates the configuration of ConfigManager and waits until the update process completes.
func (m *ConfigManager) RefreshAndWait() error {
	select {
	case err := <-m.sendRefreshSignal():
		return err
	case <-m.waitTimer():
		m.log("wait timeout")
		return iface.ErrWaitTimeout
	}
}

// waitTimer returns a valid timer channel if waitTimeout is set
func (m *ConfigManager) waitTimer() <-chan time.Time {
	if m.waitTimeout > 0 {
		return time.After(m.waitTimeout)
	} else {
		return emptyTimeChannel
	}
}

// RegisterConfigChangeListener registers a listener to config manager with a unique identifier for deregistration
func (m *ConfigManager) RegisterConfigChangeListener(identifier string, listener iface.ConfigChangeListener) {
	m.listeners.registerListener(identifier, listener)
}

// DeregisterConfigChangeListener removes a listener from config manager
func (m *ConfigManager) DeregisterConfigChangeListener(identifier string) {
	m.listeners.deregisterListener(identifier)
}

// Stop ends the ConfigManager's refreshConfigLoop
func (m *ConfigManager) Stop() {
	m.stopSignal <- struct{}{}
}

// initRefreshTicker initializes a ticker for refreshing the configuration file.
// If refreshInterval is non-zero, a new ticker is created. Otherwise, the tickerRefreshSignal remains empty.
func (m *ConfigManager) initRefreshTicker() {
	if m.refreshInterval > 0 {
		m.refreshTicker = time.NewTicker(m.refreshInterval)
		m.tickerRefreshSignal = m.refreshTicker.C
	} else {
		m.tickerRefreshSignal = emptyTimeChannel
	}
}

// refreshConfigLoop is a method that continuously refreshes the configuration by periodically loading
// and updating it as well as listening for any refresh signals.
func (m *ConfigManager) refreshConfigLoop() {
	defer func() {
		if r := recover(); r != nil {
			m.log("refreshConfigLoop panic: err=%v", r)
		}
	}()

	m.initRefreshTicker()
	defer func() {
		if m.refreshTicker != nil {
			m.refreshTicker.Stop()
		}
	}()

	for {
		waiters, stopped := m.waitForSignal()
		if stopped {
			return
		}
		err := m.loadAndUpdateConfig()
		notifyWaiters(waiters, err)
	}
}

func (m *ConfigManager) waitForSignal() (waiters []iface.RefreshCompleteSignal, stopped bool) {
	var firstWaiter iface.RefreshCompleteSignal
	select {
	case firstWaiter = <-m.manualRefreshSignal:
		waiters = append(waiters, firstWaiter)
		// collect all the waiters so that we can merge them into one refresh
		waiters = append(waiters, collectWaitersNonBlocking(m.manualRefreshSignal)...)
	case <-m.tickerRefreshSignal:
		// periodical refresh
	case <-m.stopSignal:
		return nil, true
	}
	return waiters, false
}

// collectWaitersNonBlocking collects all the waiters from the manualRefreshSignal channel in a non-blocking way.
func collectWaitersNonBlocking(refreshSignal chan iface.RefreshCompleteSignal) []iface.RefreshCompleteSignal {
	var waiters []iface.RefreshCompleteSignal
	var channel iface.RefreshCompleteSignal
	for {
		select {
		case channel = <-refreshSignal:
			waiters = append(waiters, channel)
		default:
			return waiters
		}
	}
}

func notifyWaiters(waiters []iface.RefreshCompleteSignal, err error) {
	for _, waiter := range waiters {
		// waiter has a buffer of 1, so it won't block
		waiter <- err
	}
}

// loadAndUpdateConfig updates the current configuration by loading and comparing it with the latest configuration.
// It retrieves the latest configuration with the configProvider and compares with the current configuration.
// If the configuration has changed, it updates the current configuration with the latest version and notifies listeners.
func (m *ConfigManager) loadAndUpdateConfig() error {
	currentConfig := m.shallowCopyConfig()

	latestConfig, err := m.configProvider.LoadConfig(currentConfig)
	if err != nil && err != iface.ErrNotModified {
		m.log("load config failed: err=%v", err)
		return err
	}

	if err != iface.ErrNotModified {
		// if modified, update current config & call listeners
		m.updateCurrentConfig(latestConfig)
		differences := util.CompareConfig(currentConfig, latestConfig)
		m.callConfigChangeListeners(differences)
	}
	return nil
}

func (m *ConfigManager) shallowCopyConfig() map[string]iface.ConfigValue {
	return util.ConfigSyncMapToMap(m.getCurrentConfigAtomic())
}

func (m *ConfigManager) getCurrentConfigAtomic() *sync.Map {
	return (*sync.Map)(atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(&m.currentConfig))))
}

func (m *ConfigManager) updateCurrentConfig(latestConfig map[string]iface.ConfigValue) {
	newConfig := util.ConfigMapToSyncMap(latestConfig)
	atomic.StorePointer(
		(*unsafe.Pointer)(unsafe.Pointer(&m.currentConfig)),
		unsafe.Pointer(newConfig),
	)
}

func (m *ConfigManager) callConfigChangeListeners(differences []iface.ConfigChange) {
	defer func() {
		if r := recover(); r != nil {
			m.log("callConfigChangeListeners panic: err=%v", r)
		}
	}()

	if len(differences) == 0 {
		return
	}

	for _, listener := range m.listeners.copyServiceListeners() {
		for _, change := range differences {
			listener(change)
		}
	}
}

// GetConfig retrieves the configuration value for the given key from the ConfigManager.
// If the key is not found, it returns an error indicating the key was not found.
// If the value of the key is not of type ConfigValue, it panics with an error message.
func (m *ConfigManager) GetConfig(key iface.ConfigKey) (iface.ConfigValue, error) {
	if value, ok := m.getCurrentConfigAtomic().Load(key.ToString()); ok {
		if configValue, ok := value.(iface.ConfigValue); ok {
			return configValue, nil
		}
		panic(fmt.Sprintf("GetConfigValue: invalid config Value type for %v", key.ToString())) // should not happen, fail fast
	}
	return nil, iface.ErrConfigNotFound
}

// GetConfigItem retrieves a specific value from the configuration based on the given key and item type.
// Returns the retrieved configuration value item and an error if unable to retrieve it.
func (m *ConfigManager) GetConfigItem(key iface.ConfigKey, itemType iface.ItemType) (iface.ConfigValueItem, error) {
	configValue, err := m.GetConfig(key)
	if err != nil {
		return nil, err
	}
	return configValue.GetItem(itemType)
}

// Dump exports the ConfigManager's configuration to a file specified by the file path.
func (m *ConfigManager) Dump(filepath string) error {
	if m.configSerializer == nil {
		return errors.New("no config serializer")
	}
	if data, err := m.configSerializer.Encode(m.shallowCopyConfig()); err == nil {
		return ioutil.WriteFile(filepath, data, 0o644)
	} else {
		return err
	}
}
