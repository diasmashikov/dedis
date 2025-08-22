package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/diasmashikov/dedis/internal/cache"
)

var 
( 
	port = flag.Int("port", 8002, "http port for replica")
	master = flag.String("master", "http://localhost:8001", "http port of the master")
)

func main() {
	flag.Parse()
	store := cache.New()

	http.HandleFunc("/replicate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

		if err := r.ParseForm(); err != nil {
            http.Error(w, "bad form", http.StatusBadRequest)
            return
        }

		k := r.FormValue("k")
		v := r.FormValue("v")

		store.Set(k, v)
		fmt.Fprintln(w, "ok")
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	http.HandleFunc("/get", func(w http.ResponseWriter, r *http.Request) {
        k := r.URL.Query().Get("k")
        if k == "" {
            http.Error(w, "invalid k", http.StatusBadRequest)
            return
        }
        v, ok := store.Get(k)
        if !ok {
            http.Error(w, "not found", http.StatusNotFound)
            return
        }
        fmt.Fprintln(w, v)
    })

	go func() {
		addr := fmt.Sprintf("localhost:%d", *port)
		form := url.Values{"addr": {addr}}
		client := &http.Client{Timeout: 2 * time.Second}
		_, err := client.PostForm(*master+"/register", form)
		if err != nil {
			log.Printf("replicate to %s failed: %v", addr, err)
		}
	}()

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("replica listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}