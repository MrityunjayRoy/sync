package main

import (
	"fmt"
	"os"

	"github.com/MrityunjayRoy/sync/internal/room"
)

func main() {
	fmt.Println("Starting server from cmd/server...")
	room.StartClient()
	os.Exit(0)
}
