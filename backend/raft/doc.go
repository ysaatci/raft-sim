// Package raft implements the Raft consensus algorithm as a pure,
// deterministic state machine.
//
// The core starts no goroutines and reads no clocks; its only I/O goes
// through the Storage it is given.
// Callers advance logical time with Tick, deliver messages with Step and
// collect outbound messages, state changes and committed entries from Ready.
// Randomness (election timeouts) comes from a seeded source supplied in the
// configuration, so a run is fully reproducible from its seed.
package raft
