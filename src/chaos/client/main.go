package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func makeRequest(url string, key string) {
	data := []byte(fmt.Sprintf(`{"key":"%s"}`, key))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()
}

func main() {
	key := os.Args[1]
	url := "http://localhost:8080"

	makeRequest(url, key)
	fmt.Printf("Chaos started for %s\n", key)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	makeRequest(url, key)
	fmt.Printf("\nChaos ended for %s\n", key)
}
