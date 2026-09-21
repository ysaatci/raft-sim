// Package sim runs a Raft cluster on a simulated network in virtual time.
//
// Everything is deterministic: given the same seed and the same sequence of
// actions, a simulation produces exactly the same run. This is what makes
// pause, step, rewind and shareable scenarios possible, and lets a failing
// randomized test be replayed from its seed.
package sim
