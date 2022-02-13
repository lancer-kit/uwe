// Package uwe (Ubiquitous Workers Engine) is a common toolset for building and
// organizing Go application with actor-like workers.
//
// `Chief` is a supervisor that can be placed at the top of the go application's execution stack,
// it is blocked until SIGTERM is intercepted and then it shutdown all workers gracefully.
// Also, `Chief` can be used as a child supervisor inside the` Worker`, which is launched by `Chief` at the top-level.
//
// `Worker` is an interface for async workers which launches and manages by the **Chief**.
//
// 1. `Init()` - method used to initialize some state of the worker that required interaction with outer context,
// for example, initialize some connectors. In many cases this method is optional, so it can be implemented as empty:
//  `func (*W) Init() error { return nil }`.
// 2. `Run(ctx Context) error` - starts the `Worker` instance execution. The context will provide a signal
// when a worker must stop through the `ctx.Done()`.
//
// Workers lifecycle:
//
// ```text
// (*) -> [New] -> [Initialized] -> [Run] -> [Stopped]
//          |             |           |
//          |             |           ↓
//          |-------------|------> [Failed]
// ```
package uwe
