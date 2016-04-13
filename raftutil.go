package raftutil

import (
	"fmt"
	"net/http"

	"github.com/hashicorp/raft"
)

// ProxyToLeader sends http request to raft leader
func ProxyToLeader(r *raft.Raft, req *http.Request, client *http.Client) (*http.Response, error) {
	// Get raft leader
	leader := r.Leader()

	fmt.Println("is it the leader?", leader)

	if leader == "" {
		return nil, raft.ErrNotLeader
	}

	// Proxy request to leader

	// return response
	return nil, nil
}
