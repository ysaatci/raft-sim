//go:build js && wasm

// Command wasm exposes the Raft simulator to JavaScript as a global
// `raftSim` object. Every function takes and returns plain values:
// queries return JSON strings, and mutations return null on success or
// an error message.
package main

import (
	"syscall/js"

	"github.com/ysaatci/raft-sim/backend/bridge"
)

func main() {
	var api bridge.API
	js.Global().Set("raftSim", js.ValueOf(map[string]any{
		"defaultConfig": fn(func(js.Value, []js.Value) any { return bridge.DefaultConfig() }),
		"create":        mutation(func(a []js.Value) error { return api.Create(a[0].String()) }),
		"advance":       mutation(func(a []js.Value) error { return api.Advance(a[0].Int()) }),
		"seek":          mutation(func(a []js.Value) error { return api.Seek(a[0].Int()) }),
		"do":            mutation(func(a []js.Value) error { return api.Do(a[0].String()) }),
		"state":         query(func([]js.Value) (string, error) { return api.State() }),
		"events":        query(func(a []js.Value) (string, error) { return api.Events(a[0].Int()) }),
		"actions":       query(func([]js.Value) (string, error) { return api.Actions() }),
	}))
	select {} // keep the Go runtime alive to serve calls
}

func fn(f func(js.Value, []js.Value) any) js.Func { return js.FuncOf(f) }

// mutation wraps a call that returns only an error: null on success,
// otherwise the error message.
func mutation(f func([]js.Value) error) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		if err := f(args); err != nil {
			return err.Error()
		}
		return nil
	})
}

// query wraps a call that returns JSON: the JSON string on success,
// otherwise an object {error: message}.
func query(f func([]js.Value) (string, error)) js.Func {
	return js.FuncOf(func(_ js.Value, args []js.Value) any {
		s, err := f(args)
		if err != nil {
			return map[string]any{"error": err.Error()}
		}
		return s
	})
}
