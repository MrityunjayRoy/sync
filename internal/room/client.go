package room

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)

func StartClient() {
	conn, err := net.Dial("tcp", ":9000")
	if err != nil {
		fmt.Printf("Error connecting: %v\n", err)
		return
	}

	defer conn.Close()

	fmt.Println("Connected to chat server")
	
	//Background goroutine: read from server
	go func() {
		reader := bufio.NewReader(conn)
		for {
			message, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Disconnected from server")
				os.Exit(0)
			}

			//clear current prompt line and print message
			fmt.Print("\r", message)
			fmt.Print(">> ")
		}
	}()

	// main goroutine read from stdin
	inputReader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to the server")

	for {
		fmt.Print(">> ")
		message, _ := inputReader.ReadString('\n')
		message = strings.TrimSpace(message)

		if message == "" {
			continue
		}

		conn.Write([]byte(message + "\n"))
	}
}
