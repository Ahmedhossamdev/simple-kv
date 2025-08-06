package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Ahmedhossamdev/simple-kv/peer"
	"github.com/Ahmedhossamdev/simple-kv/store"
	"github.com/google/uuid"
)

func Start(addr string, s *store.Store, peers []string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	// Start automatic sync services if we have peers
	if len(peers) > 0 {
		// Startup sync - sync when node starts
		go func() {
			time.Sleep(3 * time.Second) // Wait for server to be ready
			fmt.Println("🔄 Starting automatic startup sync...")
			performStartupSync(s, peers)
		}()

		// Periodic sync - sync every 30 seconds
		go func() {
			time.Sleep(10 * time.Second) // Wait longer for initial startup
			fmt.Println("🔄 Starting periodic sync service...")
			startPeriodicSync(s, peers)
		}()

		// Peer recovery monitor - detect when peers come back online
		go func() {
			time.Sleep(5 * time.Second)
			fmt.Println("🔍 Starting peer recovery monitor...")
			startPeerRecoveryMonitor(s, peers)
		}()
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go handleConnection(conn, s, peers)
	}
}

func handleConnection(conn net.Conn, s *store.Store, peers []string) {
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		line := scanner.Text()

		fmt.Println(line)

		mainParts := strings.Split(line, "|")
		cmdParts := strings.Fields(mainParts[0])

		if len(cmdParts) == 0 {
			continue
		}

		cmd := strings.ToUpper(cmdParts[0])

		// Extract msg-id and timestamp
		// SET X 1|msg-id:f7854c7b-9c75-486b-bf65-230717420250|ts:1754412219586286400
		msgID := ""
		timestamp := time.Now().UnixNano()
		for _, part := range mainParts[1:] {
			if strings.HasPrefix(part, "msg-id:") {
				msgID = strings.TrimPrefix(part, "msg-id:")
			} else if strings.HasPrefix(part, "ts:") {
				fmt.Sscanf(strings.TrimPrefix(part, "ts:"), "%d", &timestamp)
			}
		}

		switch cmd {
		case "SET":
			if len(cmdParts) != 3 {
				fmt.Fprintln(conn, "Usage: SET key value")
				continue
			}

			key, value := cmdParts[1], cmdParts[2]

			if msgID == "" {
				msgID = uuid.New().String()
				timestamp = time.Now().UnixNano()
				// Rebuild full message including metadata
				line = fmt.Sprintf("SET %s %s|msg-id:%s|ts:%d", key, value, msgID, timestamp)
				// Broadcast to peers
				peer.BroadcastToPeers(peers, line)
			}

			s.Set(key, value, timestamp, msgID)
			fmt.Fprintln(conn, "OK")

		case "DEL", "DELETE":
			if len(cmdParts) != 2 {
				fmt.Fprintln(conn, "Usage: DEL key")
				continue
			}

			key := cmdParts[1]

			if msgID == "" {
				msgID = uuid.New().String()
				timestamp = time.Now().UnixNano()
				line = fmt.Sprintf("DEL %s|msg-id:%s|ts:%d", key, msgID, timestamp)
				peer.BroadcastToPeers(peers, line)
			}

			s.Del(key, timestamp, msgID)
			fmt.Fprintln(conn, "DELETED")
		case "GET":
			if len(cmdParts) != 2 {
				fmt.Fprintln(conn, "Usage: GET key")
				continue
			}
			key := cmdParts[1]
			value, ok := s.Get(key)
			if ok {
				fmt.Fprintln(conn, value)
			} else {
				fmt.Fprintln(conn, "Key not found")
			}
		case "SYNC":
			// Handle data synchronization requests
			if len(cmdParts) == 1 {
				// Return our snapshot
				snapshot, err := s.GetSnapshot()
				if err != nil {
					fmt.Fprintln(conn, "ERROR: Failed to get snapshot")
					continue
				}
				fmt.Fprintln(conn, "SNAPSHOT:")
				fmt.Fprintln(conn, string(snapshot))
			} else if len(cmdParts) == 2 && cmdParts[1] == "REQUEST" {
				// Request sync from peers
				for _, peer := range peers {
					go func(peer string) {
						peerConn, err := net.Dial("tcp", peer)
						if err != nil {
							fmt.Printf("Failed to connect to peer %s for sync: %v\n", peer, err)
							return
						}
						defer peerConn.Close()

						fmt.Fprintln(peerConn, "SYNC")

						// Read the response
						scanner := bufio.NewScanner(peerConn)
						if scanner.Scan() && scanner.Text() == "SNAPSHOT:" {
							if scanner.Scan() {
								snapshotData := scanner.Text()
								if err := s.ApplySnapshot([]byte(snapshotData)); err != nil {
									fmt.Printf("Failed to apply snapshot from %s: %v\n", peer, err)
								} else {
									fmt.Printf("Successfully synced data from %s\n", peer)
								}
							}
						}
					}(peer)
				}
				fmt.Fprintln(conn, "SYNC requested from all peers")
			}
		case "STATS":
			// Return store statistics
			stats := s.GetStats()
			statsJSON, _ := json.Marshal(stats)
			fmt.Fprintln(conn, string(statsJSON))
		default:
			fmt.Fprintln(conn, "Unknown command:", cmd)
		}
	}
}

// Automatic sync functions

// performStartupSync - sync with all peers when node starts
func performStartupSync(s *store.Store, peers []string) {
	fmt.Println("📡 Performing startup sync with peers...")

	for _, peer := range peers {
		go func(peerAddr string) {
			conn, err := net.DialTimeout("tcp", peerAddr, 5*time.Second)
			if err != nil {
				fmt.Printf("⚠️ Startup sync failed with peer %s: %v\n", peerAddr, err)
				return
			}
			defer conn.Close()

			// Request snapshot
			fmt.Fprintln(conn, "SYNC")

			scanner := bufio.NewScanner(conn)
			if scanner.Scan() && scanner.Text() == "SNAPSHOT:" {
				if scanner.Scan() {
					snapshotData := scanner.Text()
					if err := s.ApplySnapshot([]byte(snapshotData)); err != nil {
						fmt.Printf("❌ Failed to apply startup snapshot from %s: %v\n", peerAddr, err)
					} else {
						fmt.Printf("✅ Successfully synced startup data from %s\n", peerAddr)
					}
				}
			}
		}(peer)
	}
}

// startPeriodicSync - sync with peers every 30 seconds
func startPeriodicSync(s *store.Store, peers []string) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Println("🔄 Running periodic sync check...")
		performSyncWithPeers(s, peers)
	}
}

// startPeerRecoveryMonitor - monitor peers and sync when they recover
func startPeerRecoveryMonitor(s *store.Store, peers []string) {
	// Keep track of peer status
	peerStatus := make(map[string]bool)

	// Initialize all peers as unknown
	for _, peer := range peers {
		peerStatus[peer] = false
	}

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, peer := range peers {
			wasDown := !peerStatus[peer]
			isUp := checkPeerHealth(peer)

			// If peer was down and is now up, trigger sync
			if wasDown && isUp {
				fmt.Printf("🔄 Peer %s recovered! Triggering sync...\n", peer)
				go performSyncWithPeer(s, peer)
			}

			peerStatus[peer] = isUp
		}
	}
}

// checkPeerHealth - check if a peer is healthy
func checkPeerHealth(peerAddr string) bool {
	conn, err := net.DialTimeout("tcp", peerAddr, 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// performSyncWithPeers - sync with all peers
func performSyncWithPeers(s *store.Store, peers []string) {
	for _, peer := range peers {
		go performSyncWithPeer(s, peer)
	}
}

// performSyncWithPeer - sync with a specific peer
func performSyncWithPeer(s *store.Store, peerAddr string) {
	conn, err := net.DialTimeout("tcp", peerAddr, 3*time.Second)
	if err != nil {
		return // Peer is down, skip silently
	}
	defer conn.Close()

	fmt.Fprintln(conn, "SYNC")

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() && scanner.Text() == "SNAPSHOT:" {
		if scanner.Scan() {
			snapshotData := scanner.Text()
			if err := s.ApplySnapshot([]byte(snapshotData)); err != nil {
				fmt.Printf("❌ Periodic sync failed with %s: %v\n", peerAddr, err)
			} else {
				fmt.Printf("✅ Periodic sync successful with %s\n", peerAddr)
			}
		}
	}
}
