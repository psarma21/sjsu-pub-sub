package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"sjsu-pub-sub/types"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ClientMap struct {
	sync.RWMutex
	Connections map[string]string
}

var (
	ActiveConns ClientMap     // global variable to store client connections
	isLeader    chan struct{} // global channel to signal leadership status
)

func registerClientHandler(w http.ResponseWriter, r *http.Request, dbClient *mongo.Client) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	username := string(body)

	db := dbClient.Database("Test")
	usersCollection := db.Collection("Users")

	count, err := usersCollection.CountDocuments(context.Background(), bson.M{"username": username}) // get all users with input username
	if err != nil {
		http.Error(w, "Error checking username", http.StatusInternalServerError)
		return
	}

	if count > 0 { // check if username exists
		fmt.Printf("Username %s already exists, logging in...\n", username)
		http.Error(w, "Username already exists", http.StatusConflict)
		return
	}

	newUser := types.User{
		Username: username,
		Groups:   []string{},
	}

	_, err = usersCollection.InsertOne(context.Background(), newUser) // insert user if not already present
	if err != nil {
		http.Error(w, "Error inserting username", http.StatusInternalServerError)
		return
	}

	fmt.Printf("Registered new user %s!\n", username)
	w.WriteHeader(http.StatusOK)
}

func getAllGroupsHandler(w http.ResponseWriter, r *http.Request, dbClient *mongo.Client) {
	fmt.Printf("Retrieving all groups...\n")

	db := dbClient.Database("Test")
	groupsCollection := db.Collection("Groups")

	var groups []types.Group

	ctx := context.TODO()

	cursor, err := groupsCollection.Find(ctx, bson.M{}) // get all groups
	if err != nil {
		http.Error(w, "Error retrieving groups", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var group types.Group
		if err := cursor.Decode(&group); err != nil {
			http.Error(w, "Error decoding group document", http.StatusInternalServerError)
			return
		}
		groups = append(groups, group)
	}

	if err := cursor.Err(); err != nil {
		http.Error(w, "Error iterating through groups", http.StatusInternalServerError)
		return
	}

	groupsJSON, err := json.Marshal(groups)
	if err != nil {
		http.Error(w, "Error marshalling groups to JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(groupsJSON)
	fmt.Printf("Retrieved all groups!\n")
}

func joinGroupHandler(w http.ResponseWriter, r *http.Request, dbClient *mongo.Client) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	group := r.Form.Get("groupname")

	fmt.Printf("Received request for username %s to join group %s...\n", username, group)

	db := dbClient.Database("Test")
	groupsCollection := db.Collection("Groups")

	count, err := groupsCollection.CountDocuments(context.Background(), bson.M{"groupname": group}) // check if group exists
	if err != nil {
		http.Error(w, "Error validating group name", http.StatusInternalServerError)
		return
	}

	if count == 0 { // check if group exists
		http.Error(w, "Group name does not exist", http.StatusNotFound)
		return
	}

	filter := bson.M{"groupname": group}

	update := bson.M{
		"$push": bson.M{
			"groupmates": username, // append user to groupmates of group
		},
	}

	_, err = groupsCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, "Error updating Groups table", http.StatusNotFound)
		return
	}

	usersCollection := db.Collection("Users")

	// can assume username has already been validated at login

	filter = bson.M{"username": username}

	update = bson.M{
		"$push": bson.M{
			"groups": group, // append groups to groups of user
		},
	}

	_, err = usersCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, "Error updating Users table", http.StatusNotFound)
		return
	}

	fmt.Printf("Username %s successfully joined group %s!\n", username, group)
	w.WriteHeader(http.StatusOK)
}

// MulticastFromServer starts the gossip from the server. The server will multicast to the first two clients, those two clients
// will gossip with all other clients.
func MulticastFromServer(connList []string, post string, randomNumber int) error {
	if len(connList) <= 2 { // at most two clients, synchronously send. no error needed
		msg := types.GossipMessage{
			Id:           randomNumber,
			Body:         post,
			ConnsToWrite: nil, // no other clients to write to
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshaling message:", err)
			return err
		}

		for i := 0; i < len(connList); i++ {
			conn, err := net.Dial("tcp", connList[i])
			if err != nil {
				fmt.Println("Error dialing client", err)
				return err
			}

			_, err = conn.Write(msgBytes)
			if err != nil {
				fmt.Println("Error sending message to a client:", err)
			}
		}

		return nil
	} else { // more than two clients, synchronously send to both but pass on all clients info
		conn0 := connList[0]
		conn1 := connList[1]

		excludedSelfConnList := connList[1:]

		msg := types.GossipMessage{
			Id:           randomNumber,
			Body:         post,
			ConnsToWrite: excludedSelfConnList,
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshaling message:", err)
			return err
		}

		conn, err := net.Dial("tcp", conn0)
		if err != nil {
			fmt.Println("Error dialing client", err)
			return err
		}

		_, err = conn.Write(msgBytes)
		if err != nil {
			fmt.Println("Error sending message to a client:", err)
			return err
		}

		excludedSelfConnList2 := append(connList[:1], connList[2:]...)

		msg = types.GossipMessage{
			Id:           randomNumber,
			Body:         post,
			ConnsToWrite: excludedSelfConnList2,
		}

		msgBytes, err = json.Marshal(msg)
		if err != nil {
			fmt.Println("Error marshaling message:", err)
			return err
		}

		conn, err = net.Dial("tcp", conn1)
		if err != nil {
			fmt.Println("Error dialing client", err)
			return err
		}

		_, err = conn.Write(msgBytes)
		if err != nil {
			fmt.Println("Error sending message to a client:", err)
			return err
		}
	}

	return nil
}

func writePostHandler(w http.ResponseWriter, r *http.Request, dbClient *mongo.Client) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	username := r.Form.Get("username")
	group := r.Form.Get("groupname")
	post := r.Form.Get("post")

	fmt.Printf("Received request for username %s to post \"%s\" in group %s...\n", username, post, group)

	db := dbClient.Database("Test")
	groupsCollection := db.Collection("Groups")

	var groupDoc types.Group
	err = groupsCollection.FindOne(context.Background(), bson.M{"groupname": group}).Decode(&groupDoc) // check if group exists
	if err != nil {
		http.Error(w, "Error validating group name", http.StatusInternalServerError)
		return
	}

	groupMates := groupDoc.GroupMates

	fullpost := types.Post{
		Author: username,
		Group:  group,
		Body:   post,
	}

	filter := bson.M{"groupname": group}

	update := bson.M{
		"$push": bson.M{
			"posts": fullpost, // append post to posts of group
		},
	}

	_, err = groupsCollection.UpdateOne(context.Background(), filter, update)
	if err != nil {
		http.Error(w, "Error updating Groups table", http.StatusNotFound)
		return
	}

	fmt.Printf("Username %s successfully posted \"%s\" in group %s!\n", username, post, group)
	w.WriteHeader(http.StatusOK)

	fmt.Println("Initiating gossip to groupmates...")

	connListToWrite := []string{} // get list of active groupmates
	for _, user := range groupMates {
		conn, ok := ActiveConns.Connections[user]
		if ok {
			connListToWrite = append(connListToWrite, conn)
		}
	}

	if len(connListToWrite) == 0 {
		fmt.Println("No clients active currently!")
		return
	}

	rand.Seed(time.Now().UnixNano()) // choose a unique id from 1-100
	randomNumber := rand.Intn(100) + 1

	for _, elem := range connListToWrite {
		fmt.Println(elem)
	}

	err = MulticastFromServer(connListToWrite, post, randomNumber) // multicast to at most 2 clients
	if err != nil {                                                // if both secondary nodes are down
		fmt.Println("Failed multicasting post to groupmates!")
		return
	}

	fmt.Println("Multicasted post to secondary clients!")
	return
}

func listenHTTP(dbClient *mongo.Client) {
	mux := http.NewServeMux()

	// Register handlers with the custom ServeMux
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) { // register a new user
		registerClientHandler(w, r, dbClient)
	})
	mux.HandleFunc("/groups", func(w http.ResponseWriter, r *http.Request) { // get all groups
		getAllGroupsHandler(w, r, dbClient)
	})
	mux.HandleFunc("/joingroup", func(w http.ResponseWriter, r *http.Request) { // join a group
		joinGroupHandler(w, r, dbClient)
	})
	mux.HandleFunc("/writepost", func(w http.ResponseWriter, r *http.Request) { // write a post to a group
		writePostHandler(w, r, dbClient)
	})

	// Start the HTTP server with the custom ServeMux
	http.ListenAndServe(":8080", mux)
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Received client connection from:", conn.RemoteAddr())

	remoteAddr := conn.RemoteAddr()

	tcpAddr, ok := remoteAddr.(*net.TCPAddr)
	if !ok {
		fmt.Println("Not a TCP connection")
		return
	}

	hostname, err := net.LookupAddr(tcpAddr.IP.String())
	if err != nil {
		fmt.Printf("Error looking up hostname: %v\n", err)
		return
	}

	fmt.Printf("Remote hostname: %s\n", hostname[0]) // Note: hostname is returned as a slice, use [0] to get the first entry

	username := ""
	port := ""
	for {
		buffer := make([]byte, 1024)
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Client %v disconnected\n", conn.RemoteAddr())
			_, ok := ActiveConns.Connections[username]
			if ok {
				ActiveConns.Lock()
				delete(ActiveConns.Connections, username) // remove client from conn list
				ActiveConns.Unlock()
			}
			fmt.Println("Updated conn list:")
			for key, value := range ActiveConns.Connections {
				fmt.Printf("Key: %s, Value: %d\n", key, value)
			}
			return
		}

		// Unmarshal the JSON data into the AuthMessage struct
		var authMsg types.AuthMessage
		err = json.Unmarshal(buffer[:n], &authMsg)
		if err != nil {
			fmt.Println("Error unmarshalling JSON:", err)
			return
		}

		fmt.Printf("Client %v sent: %s\n", conn.RemoteAddr(), authMsg)
		username = authMsg.Username
		port = authMsg.Port
		ActiveConns.Lock()
		ActiveConns.Connections[username] = hostname[0] + port // add client to conn list. store username, IP address
		ActiveConns.Unlock()
		fmt.Println("Updated conn list:")
		for key, value := range ActiveConns.Connections {
			fmt.Printf("Key: %s, Value: %d\n", key, value)
		}
	}
}

func initDB() (*mongo.Client, error) {
	var client *mongo.Client

	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")

	ctx := context.TODO()
	var err error
	client, err = mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func listenForConnections(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleLeaderMessage(conn net.Conn, myPort string) {
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return
	}

	receivedPort := string(buffer[:n])
	fmt.Println("Received leader port number:", receivedPort)

	if myPort == receivedPort {
		isLeader <- struct{}{}
	}
}

func listenForLeaderMessages(listener net.Listener, port string) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleLeaderMessage(conn, port)
	}
}

func main() {
	clientPort := flag.Int("port", 8081, "Port number for the server")
	flag.Parse()
	leaderPort := *clientPort + 1

	stringClientPort := strconv.Itoa(*clientPort)
	stringLeaderPort := strconv.Itoa(leaderPort)

	isLeader = make(chan struct{})

	ActiveConns = ClientMap{
		Connections: make(map[string]string),
	}

	dbConn, err := initDB() // initialize MongoDB connection
	if err != nil {
		fmt.Printf("Error connecting to DB: %v\n", err)
	}

	fmt.Println("Initialized DB connection...")

	conn, err := net.Dial("tcp", "34.125.114.92:8087")
	// conn, err := net.Dial("tcp", "localhost:8087")
	if err != nil {
		fmt.Println("failed to connect to gateway")
	}
	defer conn.Close()

	listener, err := net.Listen("tcp", ":"+stringClientPort) // listening for TCP connections for future gossip
	if err != nil {
		fmt.Printf("Error listening: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("TCP client erver listening on port %s...\n", stringClientPort)

	go listenForConnections(listener)

	listener2, err := net.Listen("tcp", ":"+stringLeaderPort) // listening for TCP connections for future gossip
	if err != nil {
		fmt.Printf("Error listening: %v\n", err)
		return
	}
	defer listener2.Close()

	fmt.Printf("TCP leader server listening on port %s...\n", stringLeaderPort)

	go listenForLeaderMessages(listener2, stringLeaderPort)

	go listenHTTP(dbConn)

	select {
	case <-isLeader: // if leader, start HTTP server and take in requests
		fmt.Println("Received leader election message!")
		// fmt.Println("HTTP server listening on port 8080...")
	}

	select {}
}