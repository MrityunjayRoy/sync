package room

import (
	"fmt"
	"time"
)

func NewRoom(dataDir string) (*Room, error) {
	room := &Room{
		clients:       make(map[*Client]bool),
		join:          make(chan *Client),
		leave:         make(chan *Client),
		broadcast:     make(chan string),
		listUsers:     make(chan *Client),
		directMessage: make(chan DirectMessage),
		sessions:      make(map[string]*SessionInfo),
		messages:      make([]Message, 0),
		startTime:     time.Now(),
		dataDir:       dataDir,
	}

	if err := room.loadSnapshot(); err != nil {
		fmt.Printf("Failed to load the snapshot: %v\n", err)
	}

	if err := room.initializePersistance(); err != nil {
		return nil, err
	}

	go room.periodicSnapshot()

	return room, nil
}

func (room *Room) periodicSnapshot() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		room.messageMu.Lock()
		messageCount := len(room.messages)
		room.messageMu.Unlock()

		if messageCount > 100 {
			if err := room.createSnapshot(); err != nil {
				fmt.Printf("Snapshot failed: %v\n", err)
			}
		}
	}
}
