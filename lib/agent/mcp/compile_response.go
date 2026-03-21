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
