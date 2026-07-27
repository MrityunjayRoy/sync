package main

import (
	"fmt"
	"github.com/MrityunjayRoy/sync/internal/room"
)

func main() {
	fmt.Println("Starting client from cmd/client...")
	room.StartClient()
}
