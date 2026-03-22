// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"reflect"
	"testing"
)

func TestSplitCommandLine(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{
			`hook attach --attach kprobe:tcp_connect --source 'int x=0;'`,
			[]string{"hook", "attach", "--attach", "kprobe:tcp_connect", "--source", "int x=0;"},
		},
		{`a b c`, []string{"a", "b", "c"}},
		{`'one two' c`, []string{"one two", "c"}},
		{`x "a\"b"`, []string{"x", `a"b`}},
		{``, []string{}},
		{`   `, []string{}},
		{`break do_filp_open`, []string{"break", "do_filp_open"}},
		{`break tcp_sendmsg --sec "sport==22"`, []string{"break", "tcp_sendmsg", "--sec", "sport==22"}},
	}
	for _, tc := range cases {
		got := splitCommandLine(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("splitCommandLine(%q)\n got %#v\nwant %#v", tc.in, got, tc.want)
		}
	}
}
