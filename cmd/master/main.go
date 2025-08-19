package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/diasmashikov/dedis/internal/cache"
)

var (
	port = flag.Int("port", 8001, "http port for master")
)

func main() {
	flag.Parse()
	store := cache.New()

	var (
		mu sync.RWMutex
		addrs []string
	)
	
	log.Printf("master starting on :%d", port)

	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed. Use post", http.StatusMethodNotAllowed)
			return 
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}

		addr := r.FormValue("addr")
		if addr == "" {
			http.Error(w, "add required addres in the form of address:port", http.StatusBadRequest)
			return
		}

		mu.Lock()
		found := false 
		for _, existingAddr := range addrs {
			if existingAddr == addr {
				found = true 
				break
			}
		}
		if !found {
			addrs = append(addrs, addr)	
			log.Printf("replica registered: %s", addr)
		}
		mu.Unlock()
	})

	http.HandleFunc("/set", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
            http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
            return
        }
        if err := r.ParseForm(); err != nil {
            http.Error(w, "bad form", http.StatusBadRequest)
            return
        }
        k := r.FormValue("k")
        v := r.FormValue("v")
        if k == "" {
            http.Error(w, "invalid k", http.StatusBadRequest)
            return
        }
		store.Set(k, v)
		fmt.Fprintln(w, v)

		mu.RLock()
		targets := append([]string(nil), addrs...)
		mu.RUnlock()
		go func(k, v string, replicas []string) {
			for _, replica := range replicas {
				go func(addr string) {
					form := url.Values{}
					form.Set("k", k)
					form.Set("v", v)
					client := &http.Client{Timeout: 2 * time.Second}
					fullAddress := fmt.Sprintf("http://%s/replicate", addr)
					_, err := client.PostForm(fullAddress, form)
					if err != nil {
						log.Printf("replicate to %s failed: %v", addr, err)
					}
				}(replica)
			}
		}(k, v, targets)
	})

	http.HandleFunc("/replicas", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		fmt.Fprintln(w, strings.Join(addrs, "\n"))
	})

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("master listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))

}