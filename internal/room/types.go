package room

import (
	"net"
	"os"
	"sync"
	"time"
)

type Message struct {
	ID        int       `json:"id"`
	From      string    `json:"from"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Channel   string    `json:"channel"`
}

type Client struct {
	conn           net.Conn
	username       string
	outgoing       chan string
	lastActive     time.Time
	messageSent    int
	messageRecv    int
	isSlowClient   bool
	reconnectToken string
	mu             sync.Mutex
}

type DirectMessage struct {
	toClient *Client
	message  string
}

type SessionInfo struct {
	Username       string
	ReconnectToken string
	LastSeen       time.Time
	CreatedAt      time.Time
}

type Room struct {
	join          chan *Client
	leave         chan *Client
	broadcast     chan string
	listUsers     chan *Client
	directMessage chan DirectMessage

	clients      map[*Client]bool
	mu           sync.Mutex
	totalMesages int
	startTime    time.Time

	messages      []Message
	messageMu     sync.Mutex
	nextMessageID int

	walFile *os.File
	walMu   sync.Mutex
	dataDir string

	sessions  map[string]*SessionInfo
	sessionMu sync.Mutex
}
