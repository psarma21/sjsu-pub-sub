package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// tellServer() tells the server to register a new user
func tellServer(email string) error {
	payload := strings.NewReader(email)

	resp, err := http.Post("http://localhost:8080/register", "text/plain", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Failed to register with %d code", resp.StatusCode)
		return err
	}

	fmt.Println("Successfully registered new user")
	return nil
}

// getLoginInfo() extracts user email from cookies
func getLoginInfo(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	email, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(email), nil
}

// store() stores new user info in the cookies file
func store(email string) error {
	file, err := os.Create("login.txt")
	if err != nil {
		return err
	}

	defer file.Close()

	writer := bufio.NewWriter(file)

	_, err = writer.WriteString(fmt.Sprintf("%s\n", email))
	if err != nil {
		return err
	}

	err = writer.Flush()
	if err != nil {
		return err
	}

	return nil
}

// newUserLogin() requests new user for email
func newUserLogin() string {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Enter email here: ")
	scanner.Scan()
	email := scanner.Text()

	return email
}

// login() logs in a user
func login() (string, error) {
	filename := "login.txt" // cookies of login info
	email := ""

	if _, err := os.Stat(filename); err == nil {
		fmt.Println("Login cookies exists")
		email, err = getLoginInfo(filename)
		if err != nil {
			return "", err
		}
	} else { // new user must enter email and be registered by server
		email = newUserLogin()
		err := store(email)
		if err != nil {
			return "", err
		}

		err = tellServer(email)

		if err != nil {
			return "", err
		}
	}

	return email, nil
}

func main() {
	email, err := login() // upon client spinning up, log in

	if err != nil {
		fmt.Println("Unable to login", err)
		return
	}

	fmt.Println("User:", email)

	// infinite for loop for users to join groups, write and receive posts
	// TODO: create long-lived connection to server? or frequent ping?
	for {
	}
}
