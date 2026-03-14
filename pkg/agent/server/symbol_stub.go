//go:build !linux

package server

import "errors"

var errListNotSupported = errors.New("source not available for kernel symbol on this platform")

func listSymbolKernel(symbol string) (string, error) {
	return "", errListNotSupported
}
