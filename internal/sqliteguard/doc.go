// Package sqliteguard holds a static check over the SQLite store code.
//
// It has no runtime code. The check lives in the test file next to this one
// so it runs with the ordinary suite, needs no linter plugin, and fails in
// the same place a developer is already looking.
package sqliteguard
