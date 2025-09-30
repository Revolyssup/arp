package main

import (
	"fmt"
	"net/http"

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
		fmt.Fprintf(w, fmt.Sprintf("{\"client_ip\": \"%s\"}", string(r.RemoteAddr)))
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

	fmt.Println("Server running on :9090")
	http.ListenAndServe(":9090", nil)
}
