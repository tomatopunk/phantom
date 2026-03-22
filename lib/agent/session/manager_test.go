// Copyright 2026 The Phantom Authors
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
//
// SPDX-License-Identifier: Apache-2.0

package session

import (
	"context"
	"testing"
)

func TestManagerGetOrCreateAndClose(t *testing.T) {
	mgr := NewManager("", nil)
	ctx := context.Background()
	s1, err := mgr.GetOrCreate(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if s1.ID != "s1" {
		t.Errorf("session id want s1 got %s", s1.ID)
	}
	s2, err := mgr.GetOrCreate(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if s2 != s1 {
		t.Error("same id should return same session")
	}
	list := mgr.List()
	if len(list) != 1 || list[0] != "s1" {
		t.Errorf("list want [s1] got %v", list)
	}
	mgr.Close("s1")
	if mgr.Get("s1") != nil {
		t.Error("after close Get should return nil")
	}
	if len(mgr.List()) != 0 {
		t.Errorf("list after close want [] got %v", mgr.List())
	}
}
