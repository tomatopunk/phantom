//go:build !linux

package server

import (
	"context"

	"github.com/tomatopunk/phantom/lib/proto"
)

func collectHostMetrics(context.Context) *proto.GetHostMetricsResponse {
	return &proto.GetHostMetricsResponse{
		ErrorMessage: "GetHostMetrics is only available when the agent runs on Linux",
	}
}

func collectTaskTree(context.Context, uint32) *proto.GetTaskTreeResponse {
	return &proto.GetTaskTreeResponse{
		ErrorMessage: "GetTaskTree is only available when the agent runs on Linux",
	}
}
