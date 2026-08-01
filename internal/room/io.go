package room

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"
)

func handleClient(conn net.Conn, room *Room) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in handleClient: %v\n", r)
		}
		conn.Close()
	}()

	// Set initial timeout for username entry
	conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	reader := bufio.NewReader(conn)

	// prompt for username and connections
	conn.Write([]byte("Enter username (or 'reconnect:<username>:<token>): \n"))

	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Failed to read username:", err)
		return
	}

	input = strings.TrimSpace(input)

	var username string
	var reconnectToken string
	var isReconnecting bool

	// parse reconnection error
	if strings.HasPrefix(input, "reconnect:") {
		parts := strings.Split(input, ":")
		if len(parts) == 3 {
			username = parts[1]
			reconnectToken = parts[2]
			isReconnecting = true
		} else {
			conn.Write([]byte("Invalid format. Use: reconnect:<username>:<token>\n"))
			return
		}
	} else {
		username = input
	}

	// Generate temp/guest name if empty
	if username == "" {
		username = fmt.Sprintf("Guest%d", rand.Intn(1000))
	}

	// validate reconnection or check for duplicate
	if isReconnecting {
		if room.validateReconnectToken(username, reconnectToken) {
			fmt.Printf("%s reconnected successfully.", username)
			conn.Write([]byte(fmt.Sprintf("Welcome Back %s", username)))
		} else {
			conn.Write([]byte("Invalid token or expired session. \n"))
			return
		}
	} else {
		// New connection - check if username is already connected
		if room.isUsernameConnected(username) {
			conn.Write([]byte("Username already connected. Use reconnect if you lost connection. \n"))
			return
		}

		// create or retrive session
		room.sessionMu.Lock()
		existingSession := room.sessions[username]
		room.sessionMu.Unlock()

		if existingSession != nil {
			token := existingSession.ReconnectToken
			msg := fmt.Sprintf("Tip: Save this token: %s\n", token)
			msg += fmt.Sprintf("To reconnect: reconnect:%s:%s\n", username, token)
			conn.Write([]byte(msg))
		} else {
			session := room.createSession(username)
			token := session.ReconnectToken
			msg := fmt.Sprintf("Your Token: %s\n", token)
			msg += fmt.Sprintf("To reconnect: reconnect:%s:%s\n", username, token)
			conn.Write([]byte(msg))
		}
	}

	client := &Client{
		conn:           conn,
		username:       username,
		outgoing:       make(chan string, 10),
		lastActive:     time.Now(),
		reconnectToken: reconnectToken,
		isSlowClient:   rand.Float64() < 0.1,
	}

	// clear timeout for other operations
	conn.SetReadDeadline(time.Time{})

	// Notify chatroom
	room.join <- client

	// Send welcome message
	welcomeMsg := buildWelcomeMsg(username)
	conn.Write([]byte(welcomeMsg))

	go readMessages(client, room)
	writeMessages(client) // blocks untill disconnect

	room.updateSessionActivity(username)
	room.leave <- client
}

func buildWelcomeMsg(username string) string {
	msg := fmt.Sprintf("Welcome, %s!\n", username)
	msg += "Commands:\n"
	msg += "  /users - List all users\n"
	msg += "  /history [N] - Show last N messages\n"
	msg += "  /msg <user> <msg> - Private message\n"
	msg += "  /token - Show your reconnect token\n"
	msg += "  /stats - Show your stats\n"
	msg += "  /quit - Leave\n"
	return msg
}

func readMessages(client *Client, room *Room) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in readMessages for %s: %v\n", client.username, r)
		}
	}()

	reader := bufio.NewReader(client.conn)

	for {
		// 5min timeout
		client.conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

		message, err := reader.ReadString('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Printf("%s timed out\n", client.username)
			} else {
				fmt.Printf("%s disconnect: %v\n", client.username, err)
			}

			return
		}

		client.markActive() // update actvity timestamp

		message = strings.TrimSpace(message)
		if message == "" {
			continue
		}

		client.mu.Lock()
		client.messageRecv++
		client.mu.Unlock()

		// process commands vs. regular readMessages()
		if strings.HasPrefix(message, "/") {
			handleCommand(client, room, message)
			continue
		}

		// Regular message
		formatted := fmt.Sprintf("[%s]: %s\n", client.username, message)
		room.broadcast <- formatted
	}
}

func writeMessages(client *Client) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Panic in writeReader for %s: %v\n", client.username, r)
		}
	}()

	writer := bufio.NewWriter(client.conn)

	for message := range client.outgoing {
		// simulate slow clients (testing mode)
		if client.isSlowClient {
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		}

		_, err := writer.WriteString(message)
		if err != nil {
			fmt.Printf("Writer error for %s\n", message)
		}

		err = writer.Flush()
		if err != nil {
			fmt.Printf("Flush error for %s: %v\n", client.username, err)
		}
	}

}

func handleCommand(client *Client, room *Room, command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "/users":
		room.listUsers <- client

	case "/stats":
		client.mu.Lock()
		stats := fmt.Sprintf("Your stats: \n")
		stats += fmt.Sprintf(" Messages sent: %d\n", client.messageSent)
		stats += fmt.Sprintf(" Messages recieved: %d\n", client.messageRecv)
		stats += fmt.Sprintf(" Last active: %s ago\n", time.Since(client.lastActive).Round(time.Second))
		client.mu.Unlock()

		select {
		case client.outgoing <- stats:
		default:
		}
case "/msg":
        if len(parts) < 3 {
            select {
            case client.outgoing <- "Usage: /msg <username> <message>\n":
            default:
            }
            return
        }

        targetUsername := parts[1]
        messageText := strings.Join(parts[2:], " ")

        targetClient := room.findUserbyUsername(targetUsername)
        if targetClient == nil {
            select {
            case client.outgoing <- fmt.Sprintf("User '%s' not found\n", targetUsername):
            default:
            }
            return
        }

        privateMsg := fmt.Sprintf("[From %s]: %s\n", client.username, messageText)
        select {
        case targetClient.outgoing <- privateMsg:
        default:
            select {
            case client.outgoing <- fmt.Sprintf("%s's inbox is full\n", targetUsername):
            default:
            }
            return
        }

        select {
        case client.outgoing <- fmt.Sprintf("Message sent to %s\n", targetUsername):
        default:
        }

    case "/history":
        count := 20
        if len(parts) > 1 {
            fmt.Sscanf(parts[1], "%d", &count)
        }
        if count > 100 {
            count = 100
        }
        room.sendHistory(client, count)

    case "/token":
        room.sessionMu.Lock()
        session := room.sessions[client.username]
        room.sessionMu.Unlock()

        if session != nil {
            msg := fmt.Sprintf("Your reconnect token:\n")
            msg += fmt.Sprintf("   reconnect:%s:%s\n", client.username, session.ReconnectToken)
            select {
            case client.outgoing <- msg:
            default:
            }
        }

    case "/quit":
        announcement := fmt.Sprintf("%s left the chat\n", client.username)
        room.broadcast <- announcement

        select {
        case client.outgoing <- "Goodbye!\n":
        default:
        }

        time.Sleep(100 * time.Millisecond)
        client.conn.Close()

    default:
        select {
        case client.outgoing <- fmt.Sprintf("Unknown: %s\n", parts[0]):
        default:
        }
    }
}
