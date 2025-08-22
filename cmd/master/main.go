package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/diasmashikov/dedis/internal/cache"
)

var (
	port = flag.Int("port", 8001, "http port for master")
	baseUrl = flag.String("base-url", "http://localhost", "base url")
)

func main() {
	flag.Parse()
	store := cache.New()
	client := &http.Client{Timeout: 2 * time.Second}

	var (
		mu sync.RWMutex
		replicaAddresses map[string]struct{}
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
		if _, ok := replicaAddresses[addr]; !ok {
			replicaAddresses[addr] = struct{}{}
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
		snapshot := make(map[string]struct{}, len(replicaAddresses))
		for address := range replicaAddresses {
			snapshot[address] = struct{}{}
		}
		mu.RUnlock()
		go func(k, v string, replicas map[string]struct{}) {
			for replica := range replicas {
				go func(addr string) {
					form := url.Values{}
					form.Set("k", k)
					form.Set("v", v)
					// Todo: come up with better URL propagation for replicas
					fullAddress := fmt.Sprintf("http://%s/replicate", addr)
					_, err := client.PostForm(fullAddress, form)
					if err != nil {
						log.Printf("replicate to %s failed: %v", addr, err)
					}
				}(replica)
			}
		}(k, v, snapshot)
	})

	http.HandleFunc("/replicas", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		for address := range replicaAddresses {
			fmt.Fprintln(w, address)
		}
	})

	http.HandleFunc("/master", func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		defer mu.RUnlock()
		addr := fmt.Sprintf("localhost:%d", *port)
		fmt.Fprintln(w, addr)
	})

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			mu.RLock()
			addrs := make([]string, 0, len(replicaAddresses))
			for a := range replicaAddresses {
				addrs = append(addrs, a)
			}
			mu.RUnlock()

			for _, a := range addrs {
				full := fmt.Sprintf("http://%s/health", a)
				resp, err := client.Get(full)
				if err != nil {
					mu.Lock()
					delete(replicaAddresses, a)
					mu.Unlock()
					log.Printf("removed replica %s: %v", a, err)
					continue
				}
				resp.Body.Close()
			}
		}
	}()

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("master listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}