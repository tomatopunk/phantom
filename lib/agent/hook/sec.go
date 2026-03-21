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

package hook

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// secOpOr is the keyword for logical OR in --sec expressions (token kind and binary op).
const secOpOr = "or"

// SecToSnippet converts a condition expression into a C snippet that returns 0 when the condition fails.
// attachPoint is used to allow socket fields (sport, dport, saddr, daddr) only for kprobe:tcp_sendmsg and kprobe:tcp_recvmsg.
// Supported: field==value, field!=value, <, <=, >, >=, and, or, not, parentheses.
func SecToSnippet(sec, attachPoint string) (string, error) {
	sec = strings.TrimSpace(sec)
	if sec == "" {
		return "", fmt.Errorf("--sec expression is empty")
	}
	allowed := allowedFieldsForAttachPoint(attachPoint)
	expr, err := parseSecExpr(sec, allowed)
	if err != nil {
		return "", err
	}
	cExpr, err := expr.toC()
	if err != nil {
		return "", err
	}
	return "if (!(" + cExpr + ")) return 0;\n\t", nil
}

// allowedFieldsForAttachPoint returns the set of field names allowed for the given attach point.
// Extra fields (e.g. sport, dport, saddr, daddr) come from the prologue registry for that symbol.
func allowedFieldsForAttachPoint(attachPoint string) map[string]string {
	base := map[string]string{
		"pid": "ev.pid", "tgid": "ev.tgid", "cpu": "ev.cpu",
		"arg0": "arg0", "arg1": "arg1", "arg2": "arg2", "arg3": "arg3", "arg4": "arg4", "arg5": "arg5",
		"ret": "ret",
	}
	symbol := strings.TrimSpace(strings.ToLower(AttachPrologueKey(attachPoint)))
	if symbol == "" {
		return base
	}
	for _, f := range ExtraFieldsForSymbol(symbol) {
		base[f] = f
	}
	return base
}

// AllowedFieldsHelp returns a string listing allowed fields for an attach point (for error messages).
func AllowedFieldsHelp(attachPoint string) string {
	allowed := allowedFieldsForAttachPoint(attachPoint)
	names := make([]string, 0, len(allowed))
	for k := range allowed {
		names = append(names, k)
	}
	return strings.Join(names, ", ")
}

// --- expression parser (recursive descent) ---

type secNode interface {
	toC() (string, error)
}

type compareNode struct {
	field string
	op    string // ==, !=, <, <=, >, >=
	value uint64
	cExpr string // resolved C expression for field
}

func (n *compareNode) toC() (string, error) {
	return fmt.Sprintf("%s %s %d", n.cExpr, n.op, n.value), nil
}

type binaryNode struct {
	op    string // "and", "or"
	left  secNode
	right secNode
}

func (n *binaryNode) toC() (string, error) {
	leftC, err := n.left.toC()
	if err != nil {
		return "", err
	}
	rightC, err := n.right.toC()
	if err != nil {
		return "", err
	}
	cOp := "&&"
	if n.op == secOpOr {
		cOp = "||"
	}
	return "(" + leftC + " " + cOp + " " + rightC + ")", nil
}

type notNode struct {
	inner secNode
}

func (n *notNode) toC() (string, error) {
	innerC, err := n.inner.toC()
	if err != nil {
		return "", err
	}
	return "(!(" + innerC + "))", nil
}

type token struct {
	kind string // "ident", "number", "op", "and", "or", "not", "lp", "rp", "eof"
	val  string
	num  uint64
}

//nolint:gocyclo,funlen // single-pass lexer: one loop for all token kinds
func lex(s string) ([]token, error) {
	s = strings.TrimSpace(s)
	var toks []token
	for i := 0; i < len(s); i++ {
		if unicode.IsSpace(rune(s[i])) {
			continue
		}
		if s[i] == '(' {
			toks = append(toks, token{kind: "lp", val: "("})
			continue
		}
		if s[i] == ')' {
			toks = append(toks, token{kind: "rp", val: ")"})
			continue
		}
		if s[i] == '=' && i+1 < len(s) && s[i+1] == '=' {
			toks = append(toks, token{kind: "op", val: "=="})
			i++
			continue
		}
		if s[i] == '!' && i+1 < len(s) && s[i+1] == '=' {
			toks = append(toks, token{kind: "op", val: "!="})
			i++
			continue
		}
		if s[i] == '<' && i+1 < len(s) && s[i+1] == '=' {
			toks = append(toks, token{kind: "op", val: "<="})
			i++
			continue
		}
		if s[i] == '>' && i+1 < len(s) && s[i+1] == '=' {
			toks = append(toks, token{kind: "op", val: ">="})
			i++
			continue
		}
		if s[i] == '<' {
			toks = append(toks, token{kind: "op", val: "<"})
			continue
		}
		if s[i] == '>' {
			toks = append(toks, token{kind: "op", val: ">"})
			continue
		}
		if unicode.IsLetter(rune(s[i])) || s[i] == '_' {
			j := i
			for j < len(s) && (unicode.IsLetter(rune(s[j])) || unicode.IsDigit(rune(s[j])) || s[j] == '_') {
				j++
			}
			word := strings.ToLower(s[i:j])
			i = j - 1
			switch word {
			case "and":
				toks = append(toks, token{kind: "and", val: "and"})
			case secOpOr:
				toks = append(toks, token{kind: secOpOr, val: secOpOr})
			case "not":
				toks = append(toks, token{kind: "not", val: "not"})
			default:
				toks = append(toks, token{kind: "ident", val: word})
			}
			continue
		}
		if s[i] == '0' && i+1 < len(s) && (s[i+1] == 'x' || s[i+1] == 'X') {
			j := i + 2
			if j >= len(s) {
				return nil, fmt.Errorf("invalid hex literal")
			}
			start := j
			for j < len(s) && isHexRune(rune(s[j])) {
				j++
			}
			if j == start {
				return nil, fmt.Errorf("invalid hex literal")
			}
			numStr := s[i:j]
			num, err := strconv.ParseUint(numStr, 0, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid hex literal %q: %w", numStr, err)
			}
			toks = append(toks, token{kind: "number", val: numStr, num: num})
			i = j - 1
			continue
		}
		if unicode.IsDigit(rune(s[i])) {
			j := i
			for j < len(s) && unicode.IsDigit(rune(s[j])) {
				j++
			}
			numStr := s[i:j]
			num, err := strconv.ParseUint(numStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid number %q: %w", numStr, err)
			}
			toks = append(toks, token{kind: "number", val: numStr, num: num})
			i = j - 1
			continue
		}
		return nil, fmt.Errorf("unexpected character %q", s[i])
	}
	toks = append(toks, token{kind: "eof"})
	return toks, nil
}

func parseSecExpr(sec string, allowed map[string]string) (secNode, error) {
	toks, err := lex(sec)
	if err != nil {
		return nil, err
	}
	p := &secParser{toks: toks, i: 0, allowed: allowed}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.cur().kind != "eof" {
		return nil, fmt.Errorf("unexpected token %q after expression", p.cur().val)
	}
	return expr, nil
}

type secParser struct {
	toks    []token
	i       int
	allowed map[string]string
}

func (p *secParser) cur() token {
	if p.i >= len(p.toks) {
		return token{kind: "eof"}
	}
	return p.toks[p.i]
}

func (p *secParser) advance() {
	if p.i < len(p.toks) {
		p.i++
	}
}

func (p *secParser) parseOr() (secNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == secOpOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: secOpOr, left: left, right: right}
	}
	return left, nil
}

func (p *secParser) parseAnd() (secNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.cur().kind == "and" {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &binaryNode{op: "and", left: left, right: right}
	}
	return left, nil
}

func (p *secParser) parseNot() (secNode, error) {
	if p.cur().kind == "not" {
		p.advance()
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &notNode{inner: inner}, nil
	}
	return p.parseCompare()
}

func (p *secParser) parseCompare() (secNode, error) {
	var expr secNode
	if p.cur().kind == "lp" {
		p.advance()
		var err error
		expr, err = p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.cur().kind != "rp" {
			return nil, fmt.Errorf("missing )")
		}
		p.advance()
	} else {
		// field op number
		if p.cur().kind != "ident" {
			return nil, fmt.Errorf("expected field name, got %q", p.cur().val)
		}
		field := p.cur().val
		p.advance()
		if p.cur().kind != "op" {
			return nil, fmt.Errorf("expected ==, !=, <, <=, >, >= after field, got %q", p.cur().val)
		}
		op := p.cur().val
		p.advance()
		if p.cur().kind != "number" {
			return nil, fmt.Errorf("--sec value must be an integer (decimal or 0x hex), got %q", p.cur().val)
		}
		value := p.cur().num
		p.advance()
		cExpr, ok := p.allowed[field]
		if !ok {
			return nil, fmt.Errorf("--sec unknown field %q (allowed for this point: %s)", field, strings.Join(sortedKeys(p.allowed), ", "))
		}
		expr = &compareNode{field: field, op: op, value: value, cExpr: cExpr}
	}
	return expr, nil
}

func isHexRune(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// sort for stable error messages
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
