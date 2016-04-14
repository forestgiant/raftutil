package raftutil

import (
	"net"
	"net/http"
	"os"
	"time"

	"github.com/hashicorp/raft"
)

// RaftUtil struct
type RaftUtil struct {
	Raft *raft.Raft
}

// ProxyToLeader sends http request to raft leader
func (ru *RaftUtil) ProxyToLeader(port string, req *http.Request, client *http.Client) (*http.Response, error) {
	return ProxyToLeader(ru.Raft, port, req, client)
}

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

// CreateTransport creates a tcp raft Transport from the address supplied
func CreateTransport(raftAddress string) (raft.Transport, error) {
	// Setup Raft communication.
	addr, err := net.ResolveTCPAddr("tcp", raftAddress)
	if err != nil {
		return nil, err
	}
	return raft.NewTCPTransport(raftAddress, addr, 3, 10*time.Second, os.Stderr)
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
