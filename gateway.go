package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

var (
	activeNodes   []string // List to store active nodes' hostnames
	activeNodesMu sync.Mutex
	leaderNode    string // Variable to store the leader node's hostname
	leaderMu      sync.Mutex
)

func main() {

	runLeaderElection() // run leader election once to start

	go startServerListener() // detects new servers

	go detectCrashedPort() // detects crashed servers

	router := mux.NewRouter()

	router.HandleFunc("/{service}", handleRequest)

	fmt.Println("Gateway server listening on port 8080...")
	http.ListenAndServe(":8080", router) // HTTP router

	select {}
}

func runLeaderElection() {
	leaderHostname := electLeader()
	leaderMu.Lock()
	leaderNode = leaderHostname
	leaderMu.Unlock()

	multicastLeader(leaderHostname)
}

func startServerListener() {
	listener, err := net.Listen("tcp", ":8087") // listen for connections from servers
	if err != nil {
		fmt.Printf("Error listening for server connections: %v\n", err)
		return
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection from server: %v\n", err)
			continue
		}
		fmt.Println("Received connection:", conn.RemoteAddr().String())
		handleServerConnection(conn)
	}
}

func handleServerConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String() // get the remote address (hostname + port)
	hostname, _, _ := net.SplitHostPort(remoteAddr)

	activeNodesMu.Lock()
	activeNodes = append(activeNodes, hostname) // Add hostname to activeNodes list
	activeNodesMu.Unlock()

	runLeaderElection() // run leader election after new node comes up

	fmt.Println("Active nodes:", activeNodes)
}

func detectCrashedPort() { // check for crash every 5 seconds
	for {
		for _, hostname := range activeNodes {
			_, err := net.DialTimeout("tcp", hostname+":8082", 1*time.Second) // check for crashed port by pinging them
			if err != nil {
				activeNodesMu.Lock()
				for i, node := range activeNodes {
					if node == hostname {
						activeNodes = append(activeNodes[:i], activeNodes[i+1:]...) // remove the crashed node from activeNodes list
						break
					}
				}
				activeNodesMu.Unlock()

				runLeaderElection() // trigger leader election when a node goes down

				fmt.Println("Server", hostname, "went down, triggering leader election...")
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
			leaderHostname = hostname // choose the node with the highest hostname as the leader
		}
	}

	return leaderHostname
}

func multicastLeader(leaderHostname string) { // inform all servers who leader is
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

func handleRequest(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service := vars["service"]

	for _, node := range activeNodes {
		if node == leaderNode { // forward request to leader and write response to client
			if err := forwardRequestAndListen(service, w, r); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else { // fire and forget to all other nodes (replication)
			target := fmt.Sprintf("http://%s:8080/%s", node, service)
			forwardRequestAndForget(target, w, r)
		}
	}
}

func forwardRequestAndListen(service string, w http.ResponseWriter, r *http.Request) error {
	backendURL := fmt.Sprintf("http://%s:8080/%s", leaderNode, service)

	req, err := http.NewRequest(r.Method, backendURL, r.Body)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header = r.Header

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to backend server: %v", err)
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	w.WriteHeader(resp.StatusCode)

	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to copy response body from backend server: %v", err)
	}

	return nil
}

func forwardRequestAndForget(targetURL string, w http.ResponseWriter, r *http.Request) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	r.Host = target.Host

	proxy.ServeHTTP(w, r)

	return
}
