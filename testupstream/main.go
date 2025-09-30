package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow all origins
}

func main() {
	http.HandleFunc("/headers", func(w http.ResponseWriter, r *http.Request) {
		for name, values := range r.Header {
			for _, value := range values {
				fmt.Fprintf(w, "%s: %s\n", name, value)
			}
		}
		fmt.Fprintf(w, "httpbin")
	})

	http.HandleFunc("/ip", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"client_ip": "%s"}`, r.RemoteAddr)
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println("upgrade error:", err)
			return
		}
		defer conn.Close()

		for {
			mt, msg, err := conn.ReadMessage()
			if err != nil {
				fmt.Println("read error:", err)
				break
			}

			// If client sends "ping", respond with "pong"
			if string(msg) == "ping" {
				err = conn.WriteMessage(mt, []byte("pong"))
			} else {
				// echo other messages
				err = conn.WriteMessage(mt, msg)
			}
			if err != nil {
				fmt.Println("write error:", err)
				break
			}
		}
	})
	http.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming not supported", http.StatusInternalServerError)
			return
		}

		for i := 1; i <= 10; i++ {
			fmt.Fprintf(w, "Chunk %d at %s\n", i, time.Now().Format("15:04:05"))
			flusher.Flush()
			time.Sleep(1 * time.Second)
		}
	})

	// Large response body
	http.HandleFunc("/large", func(w http.ResponseWriter, r *http.Request) {
		sizeStr := r.URL.Query().Get("size")
		size := 1024 * 1024 // 1MB default
		if sizeStr != "" {
			if parsedSize, err := strconv.Atoi(sizeStr); err == nil {
				size = parsedSize
			}
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(size))

		// Generate repetitive data
		data := make([]byte, min(size, 8192)) // 8KB chunks
		for i := range data {
			data[i] = byte(i % 256)
		}

		written := 0
		for written < size {
			chunkSize := min(len(data), size-written)
			w.Write(data[:chunkSize])
			written += chunkSize
		}
	})
	fmt.Println("Server running on :9090")
	http.ListenAndServe(":9090", nil)
}
