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

package server

import (
	"testing"
)

func TestSessionQuotaAllowBreak(t *testing.T) {
	q := NewSessionQuota(2, 0, 0)
	if !q.AllowBreak("s1") {
		t.Error("first break should be allowed")
	}
	if !q.AllowBreak("s1") {
		t.Error("second break should be allowed")
	}
	if q.AllowBreak("s1") {
		t.Error("third break should be denied")
	}
	q.RemoveBreak("s1")
	if !q.AllowBreak("s1") {
		t.Error("after remove one break should be allowed again")
	}
}
