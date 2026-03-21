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

package mcp

import (
	"fmt"
	"strings"

	"github.com/tomatopunk/phantom/lib/proto"
	"google.golang.org/protobuf/encoding/protojson"
)

// MarshalCompileAndAttachResult returns JSON text for a successful logical response, or an error if ok is false.
func MarshalCompileAndAttachResult(resp *proto.CompileAndAttachResponse) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("compile_and_attach: nil response")
	}
	if !resp.GetOk() {
		msg := strings.TrimSpace(resp.GetErrorMessage())
		if msg == "" {
			return "", fmt.Errorf("compile_and_attach failed")
		}
		return "", fmt.Errorf("%s", msg)
	}
	b, err := protojson.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("marshal compile response: %w", err)
	}
	return string(b), nil
}
