package raftutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/forestgiant/portutil"
	"github.com/hashicorp/raft"
)

// ProxyToLeader sends http request to raft leader
func ProxyToLeader(r *raft.Raft, port string, req *http.Request, client *http.Client) (*http.Response, error) {
	// Get raft leader
	leader := r.Leader()
	if leader == "" {
		return nil, raft.ErrNotLeader
	}

	fmt.Println("LEADER!!!", leader)

	// setup proxy URL
	proxyURL := req.URL

	// Replace port
	newURL, err := portutil.ReplacePortInAddr(leader, port)
	if err != nil {
		return nil, err
	}

	// Set proxyURL host
	proxyURL.Host = newURL

	// Proxy request to leader
	return client.Do(req)
}

// TCPTransport creates a tcp raft Transport from the address supplied
func TCPTransport(raftAddress string) (raft.Transport, error) {
	// Setup Raft communication.
	addr, err := net.ResolveTCPAddr("tcp", raftAddress)
	if err != nil {
		return nil, err
	}
	return raft.NewTCPTransport(raftAddress, addr, 3, 10*time.Second, os.Stderr)
}

// Cleanup creates an os.Signal chan to listen for interrupts
// and if received shutsdown raft
func Cleanup(r *raft.Raft, raftDir string) chan error {
	interrupt := make(chan os.Signal)
	signal.Notify(interrupt, os.Interrupt)

	errc := make(chan error)

	go func() {
		<-interrupt
		fmt.Println("\nCleanup raft")
		f := r.Shutdown()
		errc <- f.Error()

		errc <- RemoveRaftFiles(raftDir)
	}()

	return errc
}

// Bootstrap puts the node into single mode to elect itself as leader
func Bootstrap(config *raft.Config) error {
	config.EnableSingleNode = true
	config.DisableBootstrapAfterElect = false

	return nil
}

// ReadPeersJSON returns the peers from the PeerStore file
func ReadPeersJSON(path string) ([]string, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	if len(b) == 0 {
		return nil, nil
	}

	var peers []string
	dec := json.NewDecoder(bytes.NewReader(b))
	if err := dec.Decode(&peers); err != nil {
		return nil, err
	}

	return peers, nil
}

// RemoveRaftFiles removes all files in raftDir
func RemoveRaftFiles(raftDir string) error {
	// Remove peers.json
	os.Remove(filepath.Join(raftDir, "peers.json"))

	// Remove raft_stela.db
	os.RemoveAll(filepath.Join(raftDir, "raft_stela.db"))

	// Remove snapshots folder
	os.RemoveAll(filepath.Join(raftDir, "snapshots"))

	return nil
}
