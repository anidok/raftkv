package api

import (
	"encoding/json"
	"io"
	"net/http"

	raftkvraft "github.com/anidok/raftkv/pkg/raft"
	"github.com/anidok/raftkv/pkg/storage"
	"github.com/hashicorp/raft"
)

// Server represents the HTTP API server
type Server struct {
	raftNode *raftkvraft.RaftNode
	store    storage.Store
}

// NewServer creates a new HTTP API server
func NewServer(raftNode *raftkvraft.RaftNode, store storage.Store) *Server {
	return &Server{
		raftNode: raftNode,
		store:    store,
	}
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	http.HandleFunc("/v1/keys/", s.handleKey)
	http.HandleFunc("/v1/status", s.handleStatus)
	http.HandleFunc("/v1/join", s.handleJoin)
	return http.ListenAndServe(addr, nil)
}

// handleKey handles key-value operations
func (s *Server) handleKey(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/v1/keys/"):]
	if key == "" {
		http.Error(w, "key is required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGet(w, r, key)
	case http.MethodPut:
		s.handlePut(w, r, key)
	case http.MethodDelete:
		s.handleDelete(w, r, key)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGet handles GET requests
func (s *Server) handleGet(w http.ResponseWriter, r *http.Request, key string) {
	value, err := s.store.Get(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if value == nil {
		http.Error(w, "key not found", http.StatusNotFound)
		return
	}
	w.Write(value)
}

// handlePut handles PUT requests
func (s *Server) handlePut(w http.ResponseWriter, r *http.Request, key string) {
	value, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	cmd := &storage.Command{
		Op:    storage.OpSet,
		Key:   key,
		Value: value,
	}

	if err := s.raftNode.Apply(cmd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleDelete handles DELETE requests
func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request, key string) {
	cmd := &storage.Command{
		Op:  storage.OpDelete,
		Key: key,
	}

	if err := s.raftNode.Apply(cmd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// handleStatus handles status requests
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	status := struct {
		Leader string `json:"leader"`
		State  string `json:"state"`
	}{
		Leader: string(s.raftNode.Raft.Leader()),
		State:  s.raftNode.Raft.State().String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleJoin handles join requests
func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		NodeID  string `json:"node_id"`
		Address string `json:"address"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Add the node to the cluster
	future := s.raftNode.Raft.AddVoter(
		raft.ServerID(req.NodeID),
		raft.ServerAddress(req.Address),
		0,
		0,
	)

	if err := future.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
