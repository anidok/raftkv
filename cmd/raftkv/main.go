package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/anidok/raftkv/pkg/api"
	"github.com/anidok/raftkv/pkg/config"
	"github.com/anidok/raftkv/pkg/raft"
	"github.com/anidok/raftkv/pkg/storage"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "path to config file")
	storageType := flag.String("storage", "bolt", "storage type (bolt or sqlite)")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	// Create storage based on type
	var store storage.Store
	switch *storageType {
	case "bolt":
		store, err = storage.NewBoltStore(cfg.DataDir)
	case "sqlite":
		store, err = storage.NewSQLiteStore(cfg.DataDir)
	default:
		log.Fatalf("unsupported storage type: %s", *storageType)
	}
	if err != nil {
		log.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	// Create Raft node
	raftNode, err := raft.NewRaftNode(
		cfg.NodeID,
		cfg.BindAddr,
		cfg.AdvertiseAddr,
		cfg.DataDir,
		store,
	)
	if err != nil {
		log.Fatalf("failed to create raft node: %v", err)
	}
	defer raftNode.Close()

	// Bootstrap or join the cluster
	if cfg.JoinAddr == "" {
		if err := raftNode.Bootstrap(); err != nil {
			log.Fatalf("failed to bootstrap cluster: %v", err)
		}
	} else {
		if err := raftNode.Join(cfg.JoinAddr); err != nil {
			log.Fatalf("failed to join cluster: %v", err)
		}
	}

	// Create and start HTTP server
	server := api.NewServer(raftNode, store)
	go func() {
		if err := server.Start(cfg.HTTPAddr); err != nil {
			log.Fatalf("failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
}
