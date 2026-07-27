package room

import (
	"fmt"
	"time"

	"github.com/MrityunjayRoy/sync/pkg/token"
)

func (room *Room) createSession(username string) *SessionInfo {
	room.sessionMu.Lock()
	defer room.sessionMu.Unlock()

	tok := token.GenerateToken()

	session := &SessionInfo{
		Username:       username,
		ReconnectToken: tok,
		LastSeen:       time.Now(),
		CreatedAt:      time.Now(),
	}

	room.sessions[username] = session

	fmt.Printf("Created session for %s (token: %s...)\n", username, tok[:8])

	return session
}

func (room *Room) validateReconnectToken(username, token string) bool {
	room.sessionMu.Lock()
	defer room.sessionMu.Unlock()

	session, exists := room.sessions[username]
	if !exists {
		return false
	}

	if session.ReconnectToken != token {
		return false
	}

	if time.Since(session.LastSeen) > 1*time.Hour {
		delete(room.sessions, username)
		return false
	}

	session.LastSeen = time.Now()

	return true
}

func (room *Room) updateSessionActivity(username string) {
	room.sessionMu.Lock()
	defer room.sessionMu.Unlock()

	if session, exists := room.sessions[username]; exists {
		session.LastSeen = time.Now()
	}
}

func (room *Room) isUsernameConnected(username string) bool {
	room.mu.Lock()
	defer room.mu.Unlock()

	for client := range room.clients {
		if client.username == username {
			return true
		}
	}

	return false
}

func (room *Room) cleanupInactiveClients() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		room.mu.Lock()
		var toRemove []*Client

		for client := range room.clients {
			if client.isInactive(5 * time.Minute) {
				fmt.Printf("Removing inactive: %s\n", client.username)
				toRemove = append(toRemove, client)
			}
		}
		room.mu.Unlock()

		for _, client := range toRemove {
			room.leave <- client
		}
	}
}
