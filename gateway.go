package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var (
	activeNodes   []string   // List to store active nodes' hostnames
	activeNodesMu sync.Mutex // Mutex to protect concurrent access to activeNodes
	leaderNode    string     // Variable to store the leader node's hostname
	leaderMu      sync.Mutex // Mutex to protect concurrent access to leaderNode
)

func main() {
	// Run leader election once at the beginning
	runLeaderElection()

	// Start listener to receive connections from servers

	go startServerListener()

	go detectCrashedPort()

	router := mux.NewRouter()

	// Register handler function for all routes
	router.HandleFunc("/{service}", handleRequest)

	// Start the gateway server
	fmt.Println("Gateway server listening on port 8080...")
	http.ListenAndServe(":8080", router)

	select {}
}

func runLeaderElection() {
	// Run leader election once
	leaderHostname := electLeader()
	leaderMu.Lock()
	leaderNode = leaderHostname
	leaderMu.Unlock()

	// Multicast leader election message to all servers
	multicastLeader(leaderHostname)
}

func startServerListener() {
	fmt.Println("came here")
	listener, err := net.Listen("tcp", ":8087") // Listen for connections from servers
	if err != nil {
		fmt.Printf("Error listening for server connections: %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept() // Accept incoming connections
		if err != nil {
			fmt.Printf("Error accepting connection from server: %v\n", err)
			continue
		}
		fmt.Println("received connection", conn.RemoteAddr().String())
		handleServerConnection(conn)
	}
}

func handleServerConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String() // Get the remote address (hostname + port)
	ip, _, _ := net.SplitHostPort(remoteAddr)
	fmt.Println(ip)

	activeNodesMu.Lock()
	activeNodes = append(activeNodes, ip) // Add hostname to activeNodes list
	activeNodesMu.Unlock()

	runLeaderElection()

	fmt.Println("active nodes", activeNodes)
}

func detectCrashedPort() {
	for {
		for _, hostname := range activeNodes {
			_, err := net.DialTimeout("tcp", hostname+":8082", 1*time.Second) // check for crashed port by pinging them
			if err != nil {
				activeNodesMu.Lock()
				for i, node := range activeNodes {
					if node == hostname {
						activeNodes = append(activeNodes[:i], activeNodes[i+1:]...) // Remove the crashed node from activeNodes list
						break
					}
				}
				activeNodesMu.Unlock()

				runLeaderElection()

				fmt.Println("Server", hostname, "went down. Triggering leader election.")
			}
		}

		time.Sleep(5 * time.Second)
	}
}

func electLeader() string {
	activeNodesMu.Lock()
	defer activeNodesMu.Unlock()

	if len(activeNodes) == 0 {
		fmt.Println("No active nodes found")
		return ""
	}

	var leaderHostname string
	for _, hostname := range activeNodes {
		if leaderHostname == "" || strings.Compare(hostname, leaderHostname) > 0 {
			leaderHostname = hostname // Choose the node with the highest hostname as the leader
		}
	}

	return leaderHostname
}

func multicastLeader(leaderHostname string) {
	activeNodesMu.Lock()
	defer activeNodesMu.Unlock()

	for _, hostname := range activeNodes {
		conn, err := net.Dial("tcp", hostname+":8082")
		if err != nil {
			continue
		}
		defer conn.Close()

		_, err = conn.Write([]byte(leaderHostname))
		if err != nil {
			continue
		}
	}
}

// func handleRequest(w http.ResponseWriter, r *http.Request) {
// 	leaderMu.Lock()
// 	leaderHostname := leaderNode
// 	leaderMu.Unlock()

// 	if leaderHostname == "" {
// 		http.Error(w, "Leader node is not available", http.StatusInternalServerError)
// 		return
// 	}

// 	body, err := ioutil.ReadAll(r.Body)
// 	if err != nil {
// 		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
// 		return
// 	}

// 	if err := forwardRequest(service, body, w, r); err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	// Forward the HTTP request to the leader nod
// 	http.Redirect(w, r, "http://"+leaderHostname+":8080"+r.RequestURI, http.StatusFound)

// }

func handleRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service := vars["service"]

	// Forward the request to the appropriate server
	if err := forwardRequest(service, w, r); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func forwardRequest(service string, w http.ResponseWriter, r *http.Request) error {
	// Define the URL of the backend server based on the service
	backendURL := fmt.Sprintf("http://%s:8080/%s", leaderNode, service)

	// Create a new request
	req, err := http.NewRequest(r.Method, backendURL, r.Body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Copy headers from the original request to the new request
	req.Header = r.Header

	// Send the request to the backend server
	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to backend server: %v", err)
	}
	defer resp.Body.Close()

	// Copy response headers from the backend server to the response writer
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set response status code from the backend server
	w.WriteHeader(resp.StatusCode)

	// Copy response body from the backend server to the response writer
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body from backend server: %v", err)
	}

	return nil
}
