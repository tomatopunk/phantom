// Copyright 2026 The Phantom Authors
//
// SPDX-License-Identifier: Apache-2.0

// Package breakdsl compiles a small predicate-only DSL to a C fragment for kprobe/tracepoint templates.
package breakdsl

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// ToCFilter returns C statements that return 0 when the predicate is false (empty dsl = no filter).
// allowed is the set of identifiers that may appear (e.g. pid, tgid, arg0…).
// isKprobe controls whether argN reads PT_REGS_PARM* from ctx.
func ToCFilter(dsl string, allowed map[string]bool, isKprobe bool) (string, error) {
	s := strings.TrimSpace(dsl)
	if s == "" {
		return "", nil
	}
	p := parser{s: s, allowed: allowed, isKprobe: isKprobe}
	expr, err := p.parseExpr()
	if err != nil {
		return "", err
	}
	p.skipSpace()
	if p.i < len(p.s) {
		return "", fmt.Errorf("unexpected trailing input")
	}
	code, err := p.emit(expr)
	if err != nil {
		return "", err
	}
	return "if (!(" + code + ")) return 0;\n", nil
}

type parser struct {
	s        string
	i        int
	allowed  map[string]bool
	isKprobe bool
}

func (p *parser) skipSpace() {
	for p.i < len(p.s) && unicode.IsSpace(rune(p.s[p.i])) {
		p.i++
	}
}

func (p *parser) parseExpr() (*node, error) {
	return p.parseOr()
}

func (p *parser) parseOr() (*node, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		if p.matchKeyword("||") {
			right, err := p.parseAnd()
			if err != nil {
				return nil, err
			}
			left = &node{op: "||", left: left, right: right}
			continue
		}
		break
	}
	return left, nil
}

func (p *parser) parseAnd() (*node, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		p.skipSpace()
		if p.matchKeyword("&&") {
			right, err := p.parsePrimary()
			if err != nil {
				return nil, err
			}
			left = &node{op: "&&", left: left, right: right}
			continue
		}
		break
	}
	return left, nil
}

func (p *parser) parsePrimary() (*node, error) {
	p.skipSpace()
	if p.i < len(p.s) && p.s[p.i] == '(' {
		p.i++
		n, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		p.skipSpace()
		if p.i >= len(p.s) || p.s[p.i] != ')' {
			return nil, fmt.Errorf("expected )")
		}
		p.i++
		return n, nil
	}
	return p.parseCmp()
}

func (p *parser) parseCmp() (*node, error) {
	lhs, err := p.parseIdentOrNumber(false)
	if err != nil {
		return nil, err
	}
	p.skipSpace()
	var op string
	switch {
	case p.matchOp("=="):
		op = "=="
	case p.matchOp("!="):
		op = "!="
	case p.matchOp("<="):
		op = "<="
	case p.matchOp(">="):
		op = ">="
	case p.matchOp("<"):
		op = "<"
	case p.matchOp(">"):
		op = ">"
	default:
		return nil, fmt.Errorf("expected comparison operator")
	}
	p.skipSpace()
	rhs, err := p.parseIdentOrNumber(true)
	if err != nil {
		return nil, err
	}
	return &node{op: op, left: lhs, right: rhs}, nil
}

func (p *parser) parseIdentOrNumber(rhs bool) (*node, error) {
	p.skipSpace()
	if p.i >= len(p.s) {
		return nil, fmt.Errorf("unexpected end")
	}
	if unicode.IsDigit(rune(p.s[p.i])) || (p.s[p.i] == '-' && rhs) {
		start := p.i
		if p.s[p.i] == '-' {
			p.i++
		}
		for p.i < len(p.s) && unicode.IsDigit(rune(p.s[p.i])) {
			p.i++
		}
		num := p.s[start:p.i]
		if _, err := strconv.ParseInt(num, 10, 64); err != nil {
			return nil, fmt.Errorf("invalid number")
		}
		return &node{op: "lit", lit: num}, nil
	}
	start := p.i
	for p.i < len(p.s) && (unicode.IsLetter(rune(p.s[p.i])) || unicode.IsDigit(rune(p.s[p.i])) || p.s[p.i] == '_') {
		p.i++
	}
	if start == p.i {
		return nil, fmt.Errorf("expected identifier")
	}
	id := strings.ToLower(p.s[start:p.i])
	if !p.allowed[id] {
		return nil, fmt.Errorf("unknown or disallowed identifier %q", id)
	}
	return &node{op: "id", ident: id}, nil
}

func (p *parser) matchKeyword(kw string) bool {
	p.skipSpace()
	if len(p.s)-p.i < len(kw) {
		return false
	}
	if p.s[p.i:p.i+len(kw)] != kw {
		return false
	}
	// boundary
	if p.i+len(kw) < len(p.s) {
		c := p.s[p.i+len(kw)]
		if c != ' ' && c != '\t' && c != '\n' && c != '(' && c != ')' {
			return false
		}
	}
	p.i += len(kw)
	return true
}

func (p *parser) matchOp(op string) bool {
	if len(p.s)-p.i < len(op) {
		return false
	}
	if p.s[p.i:p.i+len(op)] != op {
		return false
	}
	p.i += len(op)
	return true
}

type node struct {
	op            string
	lit, ident    string
	left, right *node
}

func (p *parser) emit(n *node) (string, error) {
	switch n.op {
	case "lit":
		return n.lit, nil
	case "id":
		return p.emitIdent(n.ident)
	case "==", "!=", "<", ">", "<=", ">=":
		l, err := p.emit(n.left)
		if err != nil {
			return "", err
		}
		r, err := p.emit(n.right)
		if err != nil {
			return "", err
		}
		return "(" + l + " " + n.op + " " + r + ")", nil
	case "&&", "||":
		l, err := p.emit(n.left)
		if err != nil {
			return "", err
		}
		r, err := p.emit(n.right)
		if err != nil {
			return "", err
		}
		return "(" + l + " " + n.op + " " + r + ")", nil
	default:
		return "", fmt.Errorf("internal emit")
	}
}

func (p *parser) emitIdent(id string) (string, error) {
	switch id {
	case "pid":
		return "(__u32)(bpf_get_current_pid_tgid() >> 32)", nil
	case "tgid":
		return "(__u32)bpf_get_current_pid_tgid()", nil
	case "arg0", "arg1", "arg2", "arg3", "arg4", "arg5":
		if !p.isKprobe {
			return "", fmt.Errorf("argN not allowed in tracepoint filter")
		}
		n := id[3] - '0'
		if n > 5 {
			return "", fmt.Errorf("arg out of range")
		}
		// PT_REGS_PARM1 is first arg on kprobe
		parm := int(n) + 1
		return fmt.Sprintf("(__u64)PT_REGS_PARM%d(ctx)", parm), nil
	default:
		return "", fmt.Errorf("unsupported identifier %q", id)
	}
}
