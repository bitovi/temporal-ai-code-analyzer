package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func makeRequest(url string, key string) error {
	data := []byte(fmt.Sprintf(`{"key":"%s"}`, key))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalln("Usage: `./chaos <type>`")
	}

	key := os.Args[1]
	url := "http://localhost:8080"

	err := makeRequest(url, key)
	if err != nil {
		fmt.Printf("error making request: %s\n", err)
		return
	}
	fmt.Printf("Chaos started for %s\n", key)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	<-sigs

	makeRequest(url, key)
	fmt.Printf("\nChaos ended for %s\n", key)
}
