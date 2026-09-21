package raft

// HardState is the state a node must persist before answering any message.
type HardState struct {
	Term     uint64 `json:"term"`
	VotedFor NodeID `json:"votedFor"`
	Commit   uint64 `json:"commit"`
}

// Storage persists a node's HardState and log. The node writes through to
// it synchronously; implementations decide what "durable" means (memory for
// the simulator, disk for cluster mode).
type Storage interface {
	// InitialState returns the persisted HardState and log entries
	// (starting at index 1) to restore a node after a restart.
	InitialState() (HardState, []Entry, error)
	// SetHardState persists the HardState.
	SetHardState(HardState) error
	// Append persists entries. Any stored entries at or after
	// entries[0].Index are replaced.
	Append(entries []Entry) error
}

// MemoryStorage is an in-memory Storage. It survives a simulated crash
// because the simulator keeps it while discarding the Node.
type MemoryStorage struct {
	hs      HardState
	entries []Entry // entries[i] has Index i+1
}

func NewMemoryStorage() *MemoryStorage { return &MemoryStorage{} }

func (s *MemoryStorage) InitialState() (HardState, []Entry, error) {
	return s.hs, append([]Entry(nil), s.entries...), nil
}

func (s *MemoryStorage) SetHardState(hs HardState) error {
	s.hs = hs
	return nil
}

func (s *MemoryStorage) Append(entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}
	first := entries[0].Index
	if first == 0 || first > uint64(len(s.entries))+1 {
		panic("raft: MemoryStorage append leaves a gap")
	}
	s.entries = append(s.entries[:first-1], entries...)
	return nil
}
