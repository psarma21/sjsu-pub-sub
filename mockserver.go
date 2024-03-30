package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func registerClientHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	requestBody := string(body)
	fmt.Printf("Registered new user %s", requestBody)
	w.WriteHeader(http.StatusOK)
}

func main() {
	http.HandleFunc("/register", registerClientHandler)
	fmt.Println("Server listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
