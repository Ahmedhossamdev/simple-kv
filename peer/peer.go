package peer

import (
	"fmt"
	"net"
)

func BroadcastToPeers(peers []string, message string) {
	for _, peer := range peers {
		go func(peer string) {
			conn, err := net.Dial("tcp", peer)
			if err != nil {
				fmt.Println("Failed to connect to peer:", err)
				return
			}
			defer conn.Close()
			fmt.Fprintln(conn, message)
		}(peer)
	}
}
