package room

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func (room *Room) initializePersistance() error {
	if err := os.MkdirAll(room.dataDir, 0755); err != nil {
		return fmt.Errorf("Create data dir: %w", err)
	}

	walPath := filepath.Join(room.dataDir, "messages.wal")

	if err := room.recoverFromWAL(walPath); err != nil {
		fmt.Printf("Recovery failed: %v\n", err)
	}

	file, err := os.OpenFile(walPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open wal: %w", err)
	}

	room.walFile = file
	fmt.Printf("WAL initialized: %s\n", walPath)

	return nil
}

func (room *Room) recoverFromWAL(walPath string) error {
	file, err := os.Open(walPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No WAL found!")
			return nil
		}
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	recoverd := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var msg Message
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			fmt.Printf("Skipping corrupt line: %s\n", line)
			continue
		}

		room.messages = append(room.messages, msg)

		if msg.ID >= room.nextMessageID {
			 room.nextMessageID = msg.ID + 1
		}
		 recoverd++
	}

	fmt.Printf("Recovered %d messages\n", recoverd)

	return nil
}

func (room *Room) persistMessage(msg Message) error {
	room.walMu.Lock()
	defer room.walMu.Unlock()

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	_, err = room.walFile.Write(append(data, '\n'))
	if err != nil {
		return err
	}

	return room.walFile.Sync()
}

func (room *Room) createSnapshot() error {
	 snapshotPath := filepath.Join(room.dataDir, "snapshot.json")
	 tempPath := snapshotPath + ".tmp"

	 file, err := os.Create(tempPath)
	 if err != nil {
		 return err
	 }
	 defer file.Close()

	 room.messageMu.Lock()
	 data, err := json.MarshalIndent(room.messages, "", " ")
	 room.messageMu.Unlock()

	 if err != nil {
		 return err
	 }

	 if _, err := file.Write(data); err != nil {
		 return err
	 }

	 if err := file.Sync(); err != nil {
		 return err
	 }

	 file.Close()

	 if err := os.Rename(tempPath, snapshotPath); err != nil {
		 return err
	 }

	 fmt.Printf("Snapshot created (%d messages)\n", len(room.messages))
	 return room.truncateWAL()
}

func (room *Room) truncateWAL() error {
	room.walMu.Lock()
	defer room.walMu.Unlock()

	if room.walFile != nil {
		room.walFile.Close()
	}

	walPath := filepath.Join(room.dataDir, "messages.wal")
	file, err := os.OpenFile(walPath, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	room.walFile = file
	fmt.Printf("WAL truncated")

	return nil
}

func (room *Room) loadSnapshot() error{
	snapshotPath := filepath.Join(room.dataDir, "snapshot.json")
	file, err := os.Open(snapshotPath)
	if err != nil {
		 if os.IsNotExist(err){
			 return nil
		 }
		 return err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	room.messageMu.Lock()
	err = json.Unmarshal(data, &room.messages)
	room.messageMu.Unlock()

	if err != nil {
		return err
	}

	for _, msg := range room.messages {
		if msg.ID >= room.nextMessageID {
			room.nextMessageID = msg.ID + 1
		}
	}

	fmt.Printf("Loaded %d messages from snapshot\n", len(room.messages))

	return nil
}
