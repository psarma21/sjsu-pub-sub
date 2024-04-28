package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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

const emptyStringError = "Enter a non-empty value!"

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

	var groups []Group
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
func dialAndAuthenticate(username string) (net.Conn, error) {
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		return nil, err
	}

	fmt.Println("Connected to TCP server...")

	_, err = conn.Write([]byte(username)) // send username so server can map client with username
	if err != nil {
		return nil, err
	}

	fmt.Println("Sent server username!")
	return conn, nil
}

func receiveNewPosts(conn net.Conn) {
	data := make([]byte, 1024)
	n, err := conn.Read(data)
	if err != nil {
		fmt.Println("Error reading from server:", err)
		return
	}

	fmt.Printf("Received data from TCP client: %s\n", data[:n])
}

func main() {
	username, err := login() // upon client spinning up, log in
	if err != nil {
		fmt.Printf("Unable to login: %v\n", err)
		return
	}

	conn, err := dialAndAuthenticate(username) // dial to TCP server and send username
	if err != nil {
		fmt.Printf("Unable to connect to TCP server: %v\n", err)
		return
	}

	go receiveNewPosts(conn) // receive and print new posts from server

	for {
		err := doClientFunctionalities(username)
		if err != nil {
			fmt.Printf("Unable to perform client funcionalities: %v\n", err)
		}
		fmt.Println()
	}
}
