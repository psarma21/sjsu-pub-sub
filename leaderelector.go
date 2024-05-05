package main

import (
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

const (
	server1Port = ":8082"
	server2Port = ":8084"
	server3Port = ":8086"
)

var (
	nodeStatus = map[string]bool{
		server1Port: true,
		server2Port: true,
		server3Port: true,
	}
	statusLock sync.Mutex
)

func main() {
	runLeaderElection() // run leader election once at the beginning

	go monitorServerPorts() // continuously check health of servers

	select {}
}

func runLeaderElection() { // run leader election once
	leaderPort := electLeader(server1Port, server2Port, server3Port)
	multicastLeader(leaderPort)
	fmt.Printf("Elected leader as %d\n", leaderPort)
}

func monitorServerPorts() {
	for {
		time.Sleep(5 * time.Second) // check server ports every 5 seconds

		crashedPort := detectCrashedPort(server1Port, server2Port, server3Port) // detects if any server crashes
		if crashedPort != "" {
			fmt.Printf("Server on port %s has crashed\n", crashedPort)

			updateNodeStatus(crashedPort, false) // mark server status as down

			leaderPort := electLeader(server1Port, server2Port, server3Port) // trigger leader election if a server has crashed
			multicastLeader(leaderPort)
		} else {
			checkRecoveredNodes(server1Port, server2Port, server3Port) // checks if any crashed nodes recovered
		}
	}
}

func detectCrashedPort(ports ...string) string {
	for _, port := range ports {
		if !getNodeStatus(port) {
			continue // skip crashed nodes
		}

		conn, err := net.DialTimeout("tcp", "localhost"+port, 1*time.Second) // check for crashed port by pinging them
		if err != nil {
			return port
		}
		defer conn.Close()
	}
	return "" // no port has crashed
}

func checkRecoveredNodes(ports ...string) {
	for _, port := range ports {
		if !getNodeStatus(port) {
			conn, err := net.Dial("tcp", "localhost"+port)
			if err == nil {
				updateNodeStatus(port, true) // crashed node has recovered, update its status
				conn.Close()

				leaderPort := electLeader(server1Port, server2Port, server3Port) // trigger leader election since new node arrived
				multicastLeader(leaderPort)
				fmt.Printf("Elected as leader as %d\n", leaderPort)
			}
		}
	}
}

func getNodeStatus(port string) bool {
	statusLock.Lock()
	defer statusLock.Unlock()
	return nodeStatus[port]
}

func updateNodeStatus(port string, status bool) {
	statusLock.Lock()
	defer statusLock.Unlock()
	nodeStatus[port] = status
}

func electLeader(ports ...string) int {
	activePorts := make([]int, 0)

	for _, port := range ports {
		conn, err := net.Dial("tcp", "localhost"+port)
		if err == nil {
			conn.Close()
			portNum, _ := strconv.Atoi(port[1:])
			activePorts = append(activePorts, portNum)
		}
	}

	if len(activePorts) == 0 {
		fmt.Println("No active nodes found")
		return 0
	}

	leaderPort := activePorts[0] // choose the node with highest port number
	for _, port := range activePorts {
		if port > leaderPort {
			leaderPort = port
		}
	}

	return leaderPort
}

func multicastLeader(leaderPort int) { // multicast leader port to all nodes
	multicast(server1Port, leaderPort)
	multicast(server2Port, leaderPort)
	multicast(server3Port, leaderPort)
}

func multicast(serverPort string, leaderPort int) {
	conn, err := net.Dial("tcp", "localhost"+serverPort)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte(strconv.Itoa(leaderPort)))
	if err != nil {
		return
	}
}
