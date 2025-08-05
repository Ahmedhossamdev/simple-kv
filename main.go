package main

import (
	"log"
	"os"
	"strings"

	"github.com/Ahmedhossamdev/simple-kv/server"
	"github.com/Ahmedhossamdev/simple-kv/store"
)

func main() {
	port := "8080"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	var peers []string
	if len(os.Args) > 2 {
		peers = strings.Split(os.Args[2], ",")
	}

	s := store.New()
	log.Fatal(server.Start(":"+port, s, peers))
}
