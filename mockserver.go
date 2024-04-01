package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
)

func registerClientHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	username := string(body)

	// TODO: Check if user exists. Otherwise new user, insert user info in Users table

	if username == "pasarma" {
		// TODO change to "if already exists"
		fmt.Printf("User %s already exists, logging in...\n", username)
		http.Error(w, "Error registering new user", http.StatusConflict)
		return
	}

	fmt.Printf("Registered new user %s!\n", username) // debug statement for server
	w.WriteHeader(http.StatusOK)
}

func getAllGroupsHandler(w http.ResponseWriter, r *http.Request) {
	// TODO: Make DB call to Groups table to get all groups in JSON form

	fmt.Printf("Retrieving all groups...\n")
	fmt.Fprintf(w, "Here are the new groups")
}

func joinGroupHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	group := r.Form.Get("groupname")

	// TODO: Make DB call to insert new group for user in Users table

	fmt.Printf("Received request for username %s to join group %s...\n", username, group)
	w.WriteHeader(http.StatusOK)
}

func createGroupHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	group := r.Form.Get("groupname")

	// TODO: Make DB call to create group in Users, Groups table
	// TODO: Register user as part of group

	fmt.Printf("Received request for username %s to create group %s...\n", username, group)
	w.WriteHeader(http.StatusOK)
}

func getUserGroupsHandler(w http.ResponseWriter, r *http.Request) {
	username := ""

	// get username from header
	for key, values := range r.Header {
		if key == "Username" {
			username = values[0]
			break
		}
	}

	fmt.Printf("Retrieving groups for user %s...\n", username)

	// TODO: Make DB call to get all groups in JSON form from Users table

	fmt.Fprintf(w, "Here are the new groups")
}

func writePostHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	group := r.Form.Get("groupname")
	post := r.Form.Get("post")

	// TODO: Make DB call to store post in Users, Posts table
	// TODO: Gossip to all users in group

	fmt.Printf("Received request for username %s to post \"%s\" in group %s...\n", username, post, group)
	w.WriteHeader(http.StatusOK)
}

func listenHTTP() {
	http.HandleFunc("/register", registerClientHandler) // register a new user
	http.HandleFunc("/groups", getAllGroupsHandler)     // get all groups
	http.HandleFunc("/joingroup", joinGroupHandler)     // join a group
	http.HandleFunc("/creategroup", createGroupHandler) // create a group
	http.HandleFunc("/mygroups", getUserGroupsHandler)  // get all groups for a user
	http.HandleFunc("/writepost", writePostHandler)     // write a post to a group
	http.ListenAndServe(":8080", nil)
}

func handleConnection(conn net.Conn) {
	fmt.Println("Received client connection from:", conn.RemoteAddr())
}

func main() {
	// TODO: initialize DB connection

	go listenHTTP()

	fmt.Println("HTTP server listening on port 8080...")

	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Printf("Error listening: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("TCP server listening on port 8081...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}
