package main

import (
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "34.125.114.92:8087")
	// conn, err := net.Dial("tcp", "localhost:8087")
	if err != nil {
		fmt.Println("failed to connect to gateway")
	}
	defer conn.Close()
}
