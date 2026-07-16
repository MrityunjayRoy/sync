package room

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
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

func (room *Room) Run(){
	fmt.Println("Room started...")
	go room.cleanUpInactiveClient()

	for {
		select {
		case client := <- room.join:
			room.handleJoin(client)

		case client := <- room.leave:
			room.handleLeave(client)

		case message := <- room.broadcast:
			room.handleBroadcast(message)

		case client := <- room.listUsers:
			room.sendUserList(client)

		case dm := <- room.directMessage:
			room.handleDirectMessage(dm)
		}
	}
}

func (room *Room) Shutdown() {
	fmt.Println("\n Shutting down...")
	if err := room.createSnapshot(); err != nil {
		fmt.Println("Error creating Final snapshot: %v\n", err)
	}

	if room.walFile != nil {
		room.walFile.Close()
	}

	fmt.Println("Shutdown completed gracefully!")
}

func RunServer() {
	room, err := NewRoom("./chatdata") 
		if err != nil {
			fmt.Printf("Failed to initialize: %v\n", err)
		}
		defer room.Shutdown()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			 <- sigChan
			 fmt.Println("\n Recieved shutdown signal")
			 room.Shutdown()
			 os.Exit(0)
		}()

		go room.Run()

		listener, err := net.Listen("tcp", ":9000")
		if err != nil {
			fmt.Printf("Error starting server: %v\n", err)
		}
		defer listener.Close()

		fmt.Println("Server Started on :9000")

		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting connections: %v\n", err)
				continue
			}
			fmt.Println("New Connection from", conn.RemoteAddr())
			go handleClient(conn, room)
		}
}

