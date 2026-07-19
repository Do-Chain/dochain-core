//go:build !cgo

package helpers

// WasmVMAvailable reports whether app-level tests can execute the native WasmVM.
const WasmVMAvailable = false
