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
	"encoding/json"
	"sync"
	"testing"
)

func Test_jsonDecodeWithNumber(t *testing.T) {
	type args struct {
		b []byte
		v map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		isOK    func(v map[string]interface{}) bool
	}{
		{
			name: "test1",
			args: args{
				b: []byte(`{"a":9223372036854775807}`),
				v: make(map[string]interface{}),
			},
			wantErr: false,
			isOK: func(v map[string]interface{}) bool {
				a, ok := v["a"].(json.Number)
				if !ok {
					return false
				}
				if a.String() != "9223372036854775807" {
					return false
				}
				return true
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := JsonDecodeWithNumber(tt.args.b, &tt.args.v); (err != nil) != tt.wantErr || !tt.isOK(tt.args.v) {
				t.Errorf("JsonDecodeWithNumber() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIncreaseCounter(t *testing.T) {
	wg := sync.WaitGroup{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				IncreaseCounter()
			}
		}()
	}
	wg.Wait()
	if counter != int64(10000) {
		t.Errorf("counter should be %d, but got %d", 10000, counter)
	}
}
