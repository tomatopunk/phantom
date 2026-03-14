//go:build !linux

package server

import "errors"

var errListNotSupported = errors.New("source not available for kernel symbol on this platform")

func listSymbolKernel(_ string) (string, error) {
	return "", errListNotSupported
}

func listSymbolKernelAndDisasm(symbol, _ string) (string, error) {
	return listSymbolKernel(symbol)
}
