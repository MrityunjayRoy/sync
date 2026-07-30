package room

import (
	"strings"
	"testing"
	"time"
)

func TestBroadcast(t *testing.T) {
	room, _ := NewRoom("./testdata")
	defer room.Shutdown()

	go room.Run()

	//create mock clients
	client1 := &Client{
		username: "Alice",
		outgoing: make(chan string, 10),
	}

	client2 := &Client{
		username: "Bob",
		outgoing: make(chan string, 10),
	}

	// join clients
	room.join <- client1
	room.join <- client2
	time.Sleep(100 * time.Millisecond)

	room.broadcast <- "[Alice]: Hello!"

	// verify both recieve it
	select{
	case msg := <- client1.outgoing:
		if !strings.Contains(msg, "Hello!") {
			t.Fatal("Client1 didn't recieve correct message")
		}
	case <- time.After(1 * time.Second):
		t.Fatal("Client2 didn't recieve message")
	}

	select {
	case msg := <- client2.outgoing:
		if !strings.Contains(msg, "Hellow") {
			t.Fatal("Client2 didn't recieve correct message")
		}
	case <- time.After(1 * time.Second):
		t.Fatal("CLient2 didnt recieve message")
	}
}
