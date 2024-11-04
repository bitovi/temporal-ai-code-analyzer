package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	http.HandleFunc("/", handler)

	fmt.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Failed to start server:", err)
	}
}

var ChaosMap = make(map[string]bool)

type Chaos struct {
	Key string `json:"key"`
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		query := r.URL.Query()
		key := query.Get("key")
		chaos := ChaosMap[key]

		if chaos {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte{})
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
		return
	}

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to read request body: %s", err), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		var b Chaos
		err = json.Unmarshal(body, &b)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to unmarshal request body: %s", err), http.StatusInternalServerError)
			return
		}

		ChaosMap[b.Key] = !ChaosMap[b.Key]

		w.WriteHeader(http.StatusOK)
		w.Write([]byte{})
		return
	}
}
