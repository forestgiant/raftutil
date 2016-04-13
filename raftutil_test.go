package raftutil

import (
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-msgpack/codec"
	"github.com/hashicorp/raft"
)

func TestProxyToLeader(t *testing.T) {
	config := inmemConfig(t)
	store := raft.NewInmemStore()
	logs := store
	fsm := new(MockFSM)
	dir, snap := fileSnapTest(t)
	defer os.RemoveAll(dir)

	// defer os.RemoveAll(snap)
	peerStore := new(raft.StaticPeers)
	dir2, trans := raft.NewInmemTransport("")
	defer os.RemoveAll(dir2)

	// Enable single mode so this raft is the leader
	config.EnableSingleNode = true
	config.DisableBootstrapAfterElect = false

	r, err := raft.NewRaft(config, fsm, logs, store, snap, peerStore, trans)
	if err != nil {
		t.Fatalf("[ERR] NewRaft failed: %v", err)
	}
	defer r.Shutdown()

	// Wait for leader election
	select {
	case leader := <-r.LeaderCh():
		if !leader {
			panic("Raft LeaderCh should be true")
		}
	}

	resp, err := ProxyToLeader(r, nil, nil)
	if err != nil {
		t.Error("ProxyToLeader failed:", err, resp)
	}
}

// Raft test helper functions from
// https://github.com/hashicorp/raft/blob/master/raft_test.go
func fileSnapTest(t *testing.T) (string, *raft.FileSnapshotStore) {
	// Create a test dir
	dir, err := ioutil.TempDir("", "raft")
	if err != nil {
		t.Fatalf("err: %v ", err)
	}

	snap, err := raft.NewFileSnapshotStoreWithLogger(dir, 3, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	return dir, snap
}

// Return configurations optimized for in-memory
func inmemConfig(t *testing.T) *raft.Config {
	conf := raft.DefaultConfig()
	conf.HeartbeatTimeout = 50 * time.Millisecond
	conf.ElectionTimeout = 50 * time.Millisecond
	conf.LeaderLeaseTimeout = 50 * time.Millisecond
	conf.CommitTimeout = 5 * time.Millisecond
	// conf.Logger = newTestLogger(t)
	return conf
}

// MockFSM is an implementation of the FSM interface, and just stores
// the logs sequentially.
type MockFSM struct {
	sync.Mutex
	logs [][]byte
}

type MockSnapshot struct {
	logs     [][]byte
	maxIndex int
}

func (m *MockFSM) Apply(log *raft.Log) interface{} {
	m.Lock()
	defer m.Unlock()
	m.logs = append(m.logs, log.Data)
	return len(m.logs)
}

func (m *MockFSM) Snapshot() (raft.FSMSnapshot, error) {
	m.Lock()
	defer m.Unlock()
	return &MockSnapshot{m.logs, len(m.logs)}, nil
}

func (m *MockFSM) Restore(inp io.ReadCloser) error {
	m.Lock()
	defer m.Unlock()
	defer inp.Close()
	hd := codec.MsgpackHandle{}
	dec := codec.NewDecoder(inp, &hd)

	m.logs = nil
	return dec.Decode(&m.logs)
}

func (m *MockSnapshot) Persist(sink raft.SnapshotSink) error {
	hd := codec.MsgpackHandle{}
	enc := codec.NewEncoder(sink, &hd)
	if err := enc.Encode(m.logs[:m.maxIndex]); err != nil {
		sink.Cancel()
		return err
	}
	sink.Close()
	return nil
}

func (m *MockSnapshot) Release() {
}
