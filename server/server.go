package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/Ahmedhossamdev/kvstore/peer"
	"github.com/Ahmedhossamdev/kvstore/store"
	"github.com/google/uuid"
)

func Start(addr string, s *store.Store, peers []string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
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
		default:
			fmt.Fprintln(conn, "Unknown command:", cmd)
		}
	}
}
