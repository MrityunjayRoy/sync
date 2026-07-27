package room

import (
	"fmt"
	"strings"
	"time"
)

func (room *Room) handleBroadcast(message string) {
	// parse message metadata
	parts := strings.SplitN(message, ":", 2)
	from := "system"
	messageContent := message

	if len(parts) == 2 {
		from = strings.Trim(parts[0], "[]")
		messageContent = parts[1]
	}

	room.messageMu.Lock()
	msg := Message{
		ID:        room.nextMessageID,
		From:      from,
		Content:   messageContent,
		Timestamp: time.Now(),
		Channel:   "global",
	}
	room.nextMessageID++
	room.messages = append(room.messages, msg)
	room.messageMu.Unlock()

	// persist in WAL
	if err := room.persistMessage(msg); err != nil {
		fmt.Printf("Failed to persists: %v\n", err)
	}

	room.mu.Lock()
	clients := make([]*Client, 0, len(room.clients))
	for client := range room.clients {
		clients = append(clients, client)
	}
	room.totalMesages++
	room.mu.Unlock()

	fmt.Printf("Broadcasting to %d clients: %s", len(clients), message)

	for _, client := range clients {
		select {
		case client.outgoing <- message:
			client.mu.Lock()
			client.messageSent++
			client.mu.Unlock()
		default:
			fmt.Printf("Skipped %s (channel full) \n", client.username)
		}
	}
}

func (room *Room) handleJoin(client *Client) {
	room.mu.Lock()
	room.clients[client] = true
	room.mu.Unlock()

	client.markActive()
	fmt.Printf("%s (total: %d)\n", client.username, len(room.clients))

	room.sendHistory(client, 10)

	announce := fmt.Sprintf("*** %s joined the chat ***\n", client.username)
	room.handleBroadcast(announce)
}

func (room *Room) handleLeave(client *Client) {
	room.mu.Lock()
	if !room.clients[client] {
		room.mu.Unlock()
		return
	}

	delete(room.clients, client)
	room.mu.Unlock()

	fmt.Printf("%s left (total: %d)\n", client.username, len(room.clients))

	// close channel gracefully
	select {
	case <-client.outgoing:
	default:
		close(client.outgoing)
	}

	announce := fmt.Sprintf("*** %s left the chat ***\n", client.username)
	room.handleBroadcast(announce)
}

func (room *Room) sendHistory(client *Client, count int) {
	room.mu.Lock()
	defer room.messageMu.Unlock()

	start := len(room.messages) - count
	if start < 0 {
		start = 0
	}

	historyMsg := "Recent messages: \n"
	for i := start; 1 < len(room.messages); i++ {
		msg := room.messages[i]
		historyMsg += fmt.Sprintf("[%s]: %s\n", msg.From, msg.Content)
	}

	select {
	case client.outgoing <- historyMsg:
	default:
	}
}

func (room *Room) sendUserList(client *Client) {
	room.mu.Lock()
	defer room.mu.Unlock()

	list := "User online: \n"
	for c := range room.clients {
		status := ""
		if c.isInactive(1 * time.Minute) {
			status = "(idle)"
		}
		list += fmt.Sprintf(" - %s%s\n", c.username, status)
	}

	list += fmt.Sprintf("\nTotal Messages: %d\n", room.totalMesages)
	list += fmt.Sprintf("Uptime: %s\n", time.Since(room.startTime).Round(time.Second))

	select {
	case client.outgoing <- list:
	default:
	}
}

func (room *Room) handleDirectMessage(dm DirectMessage) {
	select {
	case dm.toClient.outgoing <- dm.message:
		dm.toClient.mu.Lock()
		dm.toClient.messageSent++
		dm.toClient.mu.Unlock()
	default:
		fmt.Printf("Couldn't deliver DM to %s\n", dm.toClient.username)
	}
}

func (room *Room) findUserbyUsername(username string) *Client {
	room.mu.Lock()
	defer room.mu.Unlock()

	for c := range room.clients {
		if username == c.username {
			return c
		}

		return nil
	}
}

func (c *Client) markActive() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastActive = time.Now()
}

func (c *Client) isInactive(timeout time.Duration) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return time.Since(c.lastActive) > timeout
}
