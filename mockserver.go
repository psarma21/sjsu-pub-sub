package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type User struct {
	Username string   `json:"username"`
	Groups   []string `json:"groups"`
}

type Group struct {
	GroupName  string   `bson:"groupname"`
	Creator    string   `bson:"creator"`
	GroupMates []string `bson:"groupmates"`
	Posts      []Post   `bson:"posts"`
}

type Post struct {
	Author string `bson:"author"`
	Group  string `bson:"group"`
	Body   string `bson:"body"`
}

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

	newUser := User{
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

	var groups []Group

	ctx := context.TODO()

	cursor, err := groupsCollection.Find(ctx, bson.M{}) // get all groups
	if err != nil {
		http.Error(w, "Error retrieving groups", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var group Group
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

	count, err := groupsCollection.CountDocuments(context.Background(), bson.M{"groupname": group}) // check if group exists
	if err != nil {
		http.Error(w, "Error validating group name", http.StatusInternalServerError)
		return
	}

	if count == 0 { // check if group exists
		http.Error(w, "Group name does not exist!", http.StatusInternalServerError)
		return
	}

	fullpost := Post{
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

	// TODO: Gossip to all groupmates

	fmt.Printf("Username %s successfully posted \"%s\" in group %s!\n", username, post, group)
	w.WriteHeader(http.StatusOK)
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
	fmt.Println("Received client connection from:", conn.RemoteAddr())
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

func main() {
	dbConn, err := initDB() // initialize MongoDB connection
	if err != nil {
		fmt.Printf("Error connecting to DB: %v\n", err)
	}

	fmt.Println("Initialized DB connection...")

	go listenHTTP(dbConn) // listening for HTTP request

	fmt.Println("HTTP server listening on port 8080...")

	listener, err := net.Listen("tcp", ":8081") // listening for TCP connections for future gossip
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
