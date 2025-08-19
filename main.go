package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
)

var (
	port = flag.Int("port", 8001, "http port")
)

func main() {
	flag.Parse()
	store := struct {
		sync.RWMutex
		m map[string]string
	}{ m: make(map[string]string)}

	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		k := r.URL.Query().Get("k")
		v := r.URL.Query().Get("v")
		
		if k == "" {
			http.Error(w, "invalid k", http.StatusBadRequest)
			return
		}
		
		store.Lock()
		store.m[k] = v
		store.Unlock()
		fmt.Fprintln(w, v)
	})

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
		k := r.URL.Query().Get("k")

		if k == "" {
			http.Error(w, "invalid k", http.StatusBadRequest)
			return
		}

		store.RLock()
		v := store.m[k]
		store.RUnlock()
		fmt.Fprintln(w, v)
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("starting in-memory data store server on %s\n", addr)
    log.Fatal(http.ListenAndServe(addr, nil))
}