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
	"sync"

	"github.com/cloudwego/configmanager/iface"
)

// listenerManager manages listeners safely with a RWMutex
// Each listener is identified by a unique string, which can be used for deregister
type listenerManager struct {
	listeners map[string]iface.ConfigChangeListener
	lock      sync.RWMutex
}

func newListenerManager() *listenerManager {
	return &listenerManager{
		listeners: make(map[string]iface.ConfigChangeListener),
	}
}

// registerListener registers a ConfigChangeListener with a uniqueID that is used to track the listener
// and notify it of any changes to the configuration.
func (m *listenerManager) registerListener(uniqueID string, listener iface.ConfigChangeListener) {
	m.lock.Lock()
	m.listeners[uniqueID] = listener
	m.lock.Unlock()
}

// deregisterListener removes a listener from the listenerManager's map of listeners using the provided uniqueID.
func (m *listenerManager) deregisterListener(uniqueID string) {
	m.lock.Lock()
	delete(m.listeners, uniqueID)
	m.lock.Unlock()
}

// copyServiceListeners returns a slice of ConfigChangeListener instances registered with the listenerManager.
func (m *listenerManager) copyServiceListeners() []iface.ConfigChangeListener {
	m.lock.RLock()
	defer m.lock.RUnlock()
	newListeners := make([]iface.ConfigChangeListener, 0, len(m.listeners))
	for _, listener := range m.listeners {
		newListeners = append(newListeners, listener)
	}
	return newListeners
}

// size returns the number of listeners managed by the listenerManager
func (m *listenerManager) size() int {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return len(m.listeners)
}
