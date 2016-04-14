package raftutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
)

// ProxyToLeader sends http request to raft leader
func ProxyToLeader(r *raft.Raft, port string, req *http.Request, client *http.Client) (*http.Response, error) {
	// Get raft leader
	leader := r.Leader()
	if leader == "" {
		return nil, raft.ErrNotLeader
	}

	// setup proxy URL
	proxyURL := req.URL

	// Replace port
	newURL, err := replacePortInAddr(leader, port)
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
		r.Shutdown()

		// Remove all raft files
		// First warn if this is your current working dir
		wd, err := os.Getwd()
		if err != nil {
			errc <- err
			return
		}

		if wd == raftDir {
			errc <- errors.New("\n Warning: Current directory is the same as raft directory. Must delete raft files manually")
			return
		}

		errc <- removeContents(raftDir)
	}()

	return errc
}

// Bootstrap puts the node into single mode to elect itself as leader
func Bootstrap(config *raft.Config) {
	config.EnableSingleNode = true
	config.DisableBootstrapAfterElect = true
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

// Helper function to quickly get the port from an addr string
func getPortFromAddr(addr string) (string, error) {
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	return port, nil
}

func replacePortInAddr(addr, newPort string) (string, error) {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(host, newPort), nil
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}

	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}
