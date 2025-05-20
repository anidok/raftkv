package raft

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anidok/raftkv/pkg/storage"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// RaftNode represents a Raft node in our distributed system
type RaftNode struct {
	Raft   *raft.Raft
	store  storage.Store
	addr   raft.ServerAddress
	nodeID string
}

// NewRaftNode creates a new Raft node
func NewRaftNode(
	nodeID string,
	bindAddr string,
	advertiseAddr string,
	dataDir string,
	store storage.Store,
) (*RaftNode, error) {
	// Create Raft configuration
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)

	// Create a logger
	logger := hclog.New(&hclog.LoggerOptions{
		Name:   nodeID,
		Output: os.Stdout,
		Level:  hclog.Debug,
	})
	config.Logger = logger

	// Create the snapshot store
	snapshots, err := raft.NewFileSnapshotStore(dataDir, 3, os.Stdout)
	if err != nil {
		return nil, fmt.Errorf("failed to create snapshot store: %w", err)
	}

	// Create the log store
	logStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "raft.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create log store: %w", err)
	}

	// Create the stable store
	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(dataDir, "stable.db"))
	if err != nil {
		return nil, fmt.Errorf("failed to create stable store: %w", err)
	}

	// Create the transport
	transport, err := raft.NewTCPTransport(
		bindAddr,
		nil,
		3,
		10*time.Second,
		os.Stdout,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create transport: %w", err)
	}

	// Create the Raft instance
	r, err := raft.NewRaft(
		config,
		&fsm{store: store},
		logStore,
		stableStore,
		snapshots,
		transport,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create raft: %w", err)
	}

	log.Printf("Created Raft node %s with address %s", nodeID, advertiseAddr)

	return &RaftNode{
		Raft:   r,
		store:  store,
		addr:   raft.ServerAddress(advertiseAddr),
		nodeID: nodeID,
	}, nil
}

// Bootstrap initializes the Raft cluster with a single node
func (n *RaftNode) Bootstrap() error {
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:      raft.ServerID(n.nodeID),
				Address: n.addr,
			},
		},
	}
	log.Printf("Bootstrapping node %s with configuration: %+v", n.nodeID, configuration)
	return n.Raft.BootstrapCluster(configuration).Error()
}

// Join adds a node to the Raft cluster
func (n *RaftNode) Join(joinAddr string) error {
	log.Printf("Node %s attempting to join cluster at %s", n.nodeID, joinAddr)

	// Convert Raft port to HTTP port
	httpAddr := strings.Replace(joinAddr, "8001", "8080", 1)
	httpAddr = strings.Replace(httpAddr, "8002", "8081", 1)
	httpAddr = strings.Replace(httpAddr, "8003", "8082", 1)

	// Create the join request
	joinReq := struct {
		NodeID  string `json:"node_id"`
		Address string `json:"address"`
	}{
		NodeID:  n.nodeID,
		Address: string(n.addr),
	}

	// Marshal the request
	data, err := json.Marshal(joinReq)
	if err != nil {
		return fmt.Errorf("failed to marshal join request: %w", err)
	}

	// Send HTTP request to join endpoint
	resp, err := http.Post(
		fmt.Sprintf("http://%s/v1/join", httpAddr),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return fmt.Errorf("failed to send join request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("join request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Wait for the join to complete
	time.Sleep(2 * time.Second)

	// Verify we're part of the cluster
	config := n.Raft.GetConfiguration()
	if err := config.Error(); err != nil {
		return fmt.Errorf("failed to get cluster configuration: %w", err)
	}

	// Check if we're in the configuration
	for _, server := range config.Configuration().Servers {
		if server.ID == raft.ServerID(n.nodeID) {
			log.Printf("Node %s successfully joined the cluster", n.nodeID)
			return nil
		}
	}

	return fmt.Errorf("node %s failed to join the cluster", n.nodeID)
}

// Apply applies a command to the Raft cluster
func (n *RaftNode) Apply(cmd *storage.Command) error {
	log.Printf("Node %s applying command: %+v", n.nodeID, cmd)

	// Marshal the command
	data, err := cmd.Marshal()
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Apply the command to the Raft cluster with a timeout
	future := n.Raft.Apply(data, 5*time.Second)
	if err := future.Error(); err != nil {
		log.Printf("Node %s failed to apply command: %v", n.nodeID, err)
		return fmt.Errorf("failed to apply command: %w", err)
	}

	// Wait for the command to be applied
	if err := future.Response(); err != nil {
		log.Printf("Node %s command application failed: %v", n.nodeID, err)
		return fmt.Errorf("failed to apply command: %w", err)
	}

	log.Printf("Node %s successfully applied command", n.nodeID)
	return nil
}

// Close closes the Raft node
func (n *RaftNode) Close() error {
	return n.Raft.Shutdown().Error()
}

// fsm implements the raft.FSM interface
type fsm struct {
	store storage.Store
}

// Apply applies a command to the FSM
func (f *fsm) Apply(logEntry *raft.Log) interface{} {
	var cmd storage.Command
	if err := cmd.Unmarshal(logEntry.Data); err != nil {
		log.Printf("FSM failed to unmarshal command: %v", err)
		return err
	}

	log.Printf("FSM applying command: op=%s, key=%s", cmd.Op, cmd.Key)

	result := f.store.Apply(&cmd)
	if result != nil {
		log.Printf("FSM command application failed: %v", result)
	} else {
		log.Printf("FSM successfully applied command")
	}
	return result
}

// Snapshot returns a snapshot of the FSM
func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
	return &fsmSnapshot{store: f.store}, nil
}

// Restore restores the FSM from a snapshot
func (f *fsm) Restore(rc io.ReadCloser) error {
	defer rc.Close()
	snapshot, err := io.ReadAll(rc)
	if err != nil {
		return err
	}
	return f.store.Restore(snapshot)
}

// fsmSnapshot implements the raft.FSMSnapshot interface
type fsmSnapshot struct {
	store storage.Store
}

// Persist persists the snapshot
func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	snapshot, err := f.store.Snapshot()
	if err != nil {
		return err
	}
	_, err = sink.Write(snapshot)
	return err
}

// Release releases the snapshot
func (f *fsmSnapshot) Release() {}
