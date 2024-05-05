package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"sjsu-pub-sub/types"
	"strconv"
	"strings"
	"sync"
	"time"
)

const emptyStringError = "Enter a non-empty value!"

type PostMap struct {
	sync.RWMutex
	posts map[int]int
}

var receivedPosts PostMap

// getGroups() gets and prints all groups
func getGroups(username string) error {
	errPrefix := "Error getting groups:"

	baseUrl := "http://localhost:8080/groups"

	req, _ := http.NewRequest("GET", baseUrl, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("%s Error sending HTTP request: %v", errPrefix, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s HTTP request error: %v", errPrefix, resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s Error reading HTTP response: %v", errPrefix, err)
	}

	var groups []types.Group
	if err := json.Unmarshal(body, &groups); err != nil {
		return fmt.Errorf("%s Error unmarshalling groups JSON: %v", errPrefix, err)
	}

	fmt.Println("Groups:")
	for _, group := range groups {
		fmt.Printf("Group Name: %s\n", group.GroupName)
		fmt.Printf("Creator: %s\n", group.Creator)
		fmt.Println("Group Mates:")
		for _, mate := range group.GroupMates {
			fmt.Printf("- %s\n", mate)
		}
		fmt.Println("Posts:")
		for _, post := range group.Posts {
			fmt.Printf("- Author: %s, Group: %s, Body: %s\n", post.Author, post.Group, post.Body)
		}
		fmt.Println("--------------------------------------------------")
	}

	fmt.Println("Successfully retrieved all groups!")
	return nil
}

// joinGroup() subscribes a user to a group, allowing them to receive all new posts
func joinGroup(username string) error {
	errPrefix := "Error joining group:"

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter a group name: ")
	scanner.Scan()
	groupName := scanner.Text()

	if groupName == "" {
		return fmt.Errorf("%s %s", errPrefix, emptyStringError)
	}

	payload := []byte(fmt.Sprintf("username=%s&groupname=%s", username, groupName))

	url := "http://localhost:8080/joingroup"

	resp, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("%s Error sending HTTP request: %v", errPrefix, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s Failed to register with %d code", errPrefix, resp.StatusCode)
	}

	fmt.Printf("Successfully joined new group %s! \n", groupName)
	return nil
}

// writeMyPost() writes a post to a group
func writeMyPost(username string) error {
	errPrefix := "Error joining group:"

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Enter a group name: ")
	scanner.Scan()
	groupName := scanner.Text()

	fmt.Print("Write a post: ")
	scanner.Scan()
	post := scanner.Text()

	if groupName == "" || post == "" {
		return fmt.Errorf("%s %s", errPrefix, emptyStringError)
	}

	payload := []byte(fmt.Sprintf("username=%s&groupname=%s&post=%s", username, groupName, post))

	url := "http://localhost:8080/writepost"

	resp, err := http.Post(url, "application/x-www-form-urlencoded", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("%s Error sending HTTP request: %v", errPrefix, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s Failed to register with %d code", errPrefix, resp.StatusCode)
	}

	fmt.Printf("Successfully wrote post \"%s\" to group %s\n", post, groupName)
	return nil
}

// doClientFunctionalities() is the handler for all user functionalities
func doClientFunctionalities(username string) error {
	errPrefix := "Error handling client functionality choice:"
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Choose a number from the following choices: \nSee all groups (1) \nJoin a group (2) \nWrite a post (3)\n")
	scanner.Scan()
	optionString := scanner.Text()

	if optionString == "" {
		return fmt.Errorf("%s %s", errPrefix, emptyStringError)
	}

	option, err := strconv.Atoi(optionString)
	if err != nil {
		return fmt.Errorf("%s %s", errPrefix, err)
	}

	if option == 1 {
		return getGroups(username)
	} else if option == 2 {
		return joinGroup(username)
	} else if option == 3 {
		return writeMyPost(username)
	} else {
		return fmt.Errorf("%s Chose invalid number %d", errPrefix, option)
	}
}

// tellServer() tells the server to register a new user
func tellServer(username string) error {
	payload := strings.NewReader(username)

	url := "http://localhost:8080/register"

	resp, err := http.Post(url, "text/plain", payload)
	if err != nil {
		return fmt.Errorf("Failed to send request: %s\n", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Printf("Successfully registered new user %s! \n\n", username)
		return nil
	} else if resp.StatusCode == http.StatusConflict {
		fmt.Printf("Successfully logged in existing user %s! \n\n", username)
		return nil
	} else {
		return fmt.Errorf("Failed to register with %d code\n", resp.StatusCode)
	}
}

// userLogin() requests user to login
func userLogin() (string, error) {
	errPrefix := "Error getting user login info:"

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter a username here: ")
	scanner.Scan()
	username := scanner.Text()

	if username == "" {
		return "", fmt.Errorf("%s %s", errPrefix, emptyStringError)
	}

	return username, nil
}

// login() logs in a user
func login() (string, error) {
	// users enter username
	username, err := userLogin()
	if err != nil {
		return "", fmt.Errorf("Error getting new user login info: %v", err)
	}

	err = tellServer(username) // server registers new users and authenticates and existing users
	if err != nil {
		return "", fmt.Errorf("Error registering new user: %v", err)
	}

	return username, nil
}

// dialAndAuthenticate() creates a long-lived TCP connection to receive new posts
func dialAndAuthenticate(username string, address string) {
	msg := types.AuthMessage{
		Username: username,
		Port:     address,
	}

	bytes, _ := json.Marshal(msg)

	server1 := "8081"
	conn, err := net.Dial("tcp", "localhost:"+server1)
	if err != nil {
		fmt.Println("Unable to connect to TCP server", err)
	} else {
		fmt.Printf("Connected to TCP server %s...\n", "localhost:"+server1)

		_, err = conn.Write(bytes) // send username and port so server can map client with username
		if err != nil {
			fmt.Println("Unable to write message", err)
		}
	}

	server2 := "8083"
	conn, err = net.Dial("tcp", "localhost:"+server2)
	if err != nil {
		fmt.Println("Unable to connect to TCP server", err)
	} else {
		fmt.Printf("Connected to TCP server %s...\n", "localhost:"+server2)

		_, err = conn.Write(bytes) // send username and port so server can map client with username
		if err != nil {
			fmt.Println("Unable to write message", err)
		}
	}

	server3 := "8085"
	conn, err = net.Dial("tcp", "localhost:"+server3)
	if err != nil {
		fmt.Println("Unable to connect to TCP server", err)
	} else {
		fmt.Printf("Connected to TCP server %s...\n", "localhost:"+server3)

		_, err = conn.Write(bytes) // send username and port so server can map client with username
		if err != nil {
			fmt.Println("Unable to write message", err)
		}
	}

	fmt.Println("Sent servers username and port!")
}

func pickRandomElements(input []string, count int) []string {
	list := make([]string, len(input))
	copy(list, input)

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(list), func(i, j int) {
		list[i], list[j] = list[j], list[i]
	})

	if count > len(list) {
		count = len(list)
	}
	return list[:count]
}

func checkServerHealth(conn net.Conn) {
	for {
		data := make([]byte, 1024)
		_, err := conn.Read(data)
		if err != nil {
			fmt.Println("Error reading from server:", err)
			conn.Close()
			panic("Server closed connection")
		}
	}
}

func handleClientConnection(conn net.Conn) {
	for {
		data := make([]byte, 1024)
		n, err := conn.Read(data)
		if err != nil {
			fmt.Println("Error reading from server:", err)
			conn.Close()
			return
		}

		data = data[:n]

		var msg types.GossipMessage
		err = json.Unmarshal([]byte(data), &msg)
		if err != nil {
			fmt.Println("Error unmarshalling data:", err)
			return
		}

		msgCount, ok := receivedPosts.posts[msg.Id]
		if ok { // seen post before
			receivedPosts.posts[msg.Id] = msgCount + 1
		} else { // new post
			fmt.Println("Post received through gossip:", msg.Body)
			receivedPosts.posts[msg.Id] = 1
		}

		if len(msg.ConnsToWrite) == 0 || receivedPosts.posts[msg.Id] >= 2 { // if no more connections to write or seen post at least twice
			continue
		}

		connsToWrite := []string{}
		if len(msg.ConnsToWrite) > 4 {
			connsToWrite = pickRandomElements(msg.ConnsToWrite, 4) // pick 4 random clients to gossip to and delegate gossip to them
		} else {
			connsToWrite = msg.ConnsToWrite
		}

		for _, nextConn := range connsToWrite { // gossip to 4 other clients. ignore error
			conn, err = net.Dial("tcp", nextConn)
			if err != nil {
				fmt.Println("Error dialing client", err)
				continue
			}

			msgBytes, err := json.Marshal(msg)
			if err != nil {
				fmt.Println("Error marshaling message:", err)
				continue
			}

			_, err = conn.Write(msgBytes)
			if err != nil {
				fmt.Println("Error sending message to a client:", err)
				continue
			}
		}
	}
}

func listenForOtherClientConnections(listener net.Listener) {
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		go handleClientConnection(conn)
	}
}

func createListener() (n net.Listener, s string, err error) {
	network := "tcp"
	minPort := 5000
	maxPort := 10000

	// Generate a random port number within the specified range
	rand.Seed(time.Now().UnixNano())
	port := rand.Intn(maxPort-minPort+1) + minPort
	address := fmt.Sprintf(":%d", port)

	listener, err := net.Listen(network, address)
	if err != nil {
		fmt.Println("Client unable to start listener:", err)
		return nil, "", err
	}

	return listener, address, nil
}

func main() {
	username, err := login() // upon client spinning up, log in
	if err != nil {
		fmt.Printf("Unable to login: %v\n", err)
		return
	}

	listener, address, err := createListener()
	if err != nil {
		fmt.Printf("Unable to create listener: %v\n", err)
		return
	}

	fmt.Printf("Client is listening on port %v\n", address)

	go listenForOtherClientConnections(listener) // accept client connections and receive gossip

	dialAndAuthenticate(username, address) // dial to all TCP servers and send username

	receivedPosts = PostMap{
		posts: make(map[int]int),
	}

	for {
		err := doClientFunctionalities(username)
		if err != nil {
			fmt.Printf("Unable to perform client funcionalities: %v\n", err)
		}
		fmt.Println()
	}
}
