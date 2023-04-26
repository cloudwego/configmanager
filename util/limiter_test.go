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

package util

import (
	"testing"
	"time"
)

func TestFrequencyLimiter_Allow(t *testing.T) {
	t.Run("test1-allowed-1", func(t *testing.T) {
		l := NewFrequencyLimiter(time.Second, 1)
		defer l.Stop()
		if got := l.Allow(); got != true {
			t.Errorf("Allow() = %v, want %v", got, true)
		}
		if got := l.Allow(); got != false {
			t.Errorf("Allow() = %v, want %v", got, false)
		}
	})

	t.Run("test2-allowed-2", func(t *testing.T) {
		l := NewFrequencyLimiter(time.Second, 2)
		defer l.Stop()
		if got := l.Allow(); got != true {
			t.Errorf("Allow() = %v, want %v", got, true)
		}
		if got := l.Allow(); got != true {
			t.Errorf("Allow() = %v, want %v", got, true)
		}
		if got := l.Allow(); got != false {
			t.Errorf("Allow() = %v, want %v", got, false)
		}
	})

	t.Run("test3-allowed-0", func(t *testing.T) {
		l := NewFrequencyLimiter(time.Second, 0)
		defer l.Stop()
		if got := l.Allow(); got != false {
			t.Errorf("Allow() = %v, want %v", got, false)
		}
	})

	t.Run("test4-panic", func(t *testing.T) {
		defer func() {
			if err := recover(); err == nil {
				t.Errorf("NewFrequencyLimiter() did not panic")
			}
		}()
		_ = NewFrequencyLimiter(0, 0)
	})

	t.Run("test5-refresh", func(t *testing.T) {
		l := NewFrequencyLimiter(100*time.Millisecond, 1)
		defer l.Stop()
		if got := l.Allow(); got != true {
			t.Errorf("Allow() = %v, want %v", got, true)
		}
		if got := l.Allow(); got != false {
			t.Errorf("Allow() = %v, want %v", got, false)
		}
		time.Sleep(200 * time.Millisecond)
		if got := l.Allow(); got != true {
			t.Errorf("Allow() = %v, want %v", got, true)
		}
	})
}
