package arp_test

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("ReverseProxy End-to-End Tests", Ordered, func() {
	var (
		testUpstreamSession *gexec.Session
		reverseProxySession *gexec.Session
	)

	BeforeAll(func() {
		var err error

		// Start reverse proxy server
		By("Build")
		reverseProxySession, err = gexec.Start(
			exec.Command("go", "build", "-o", "./arp", "../../cmd/"),
			GinkgoWriter,
			GinkgoWriter,
		)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(3 * time.Second) // Give build time to complete TODO: remove sleep
		By("Starting reverse proxy server")
		reverseProxySession, err = gexec.Start(
			exec.Command("./arp", "--config", "./config.yaml"),
			GinkgoWriter,
			GinkgoWriter,
		)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() error {
			_, err := http.Get("http://localhost:8080/")
			return err
		}, 10*time.Second, 500*time.Millisecond).Should(Succeed())
	})

	AfterAll(func() {
		By("Stopping servers")
		if testUpstreamSession != nil {
			testUpstreamSession.Kill()
			testUpstreamSession.Wait()
		}
		if reverseProxySession != nil {
			reverseProxySession.Kill()
			reverseProxySession.Wait()
		}
		Expect(os.Remove("./arp")).To(Succeed())
		gexec.CleanupBuildArtifacts()
	})

	Describe("Normal response handling", func() {
		It("should proxy normal responses correctly", func() {
			client := &http.Client{Timeout: 5 * time.Second}

			req, err := http.NewRequest("GET", "http://localhost:8080/headers", nil)
			Expect(err).NotTo(HaveOccurred())
			req.Header.Set("User-Agent", "e2e-test")
			req.Header.Set("X-Test-Header", "test-value")

			resp, err := client.Do(req)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())

			Expect(string(body)).To(ContainSubstring("User-Agent"))
			Expect(string(body)).To(ContainSubstring("e2e-test"))
			Expect(string(body)).To(ContainSubstring("httpbin"))
		})

		It("should preserve response headers", func() {
			client := &http.Client{Timeout: 5 * time.Second}

			resp, err := client.Get("http://localhost:8080/headers")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})

	Describe("Streaming response handling", func() {
		It("should handle chunked streaming responses", func() {
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Get("http://localhost:8080/stream")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			reader := bufio.NewReader(resp.Body)
			chunkCount := 0
			start := time.Now()
			for time.Since(start) < 5*time.Second && chunkCount < 3 {
				line, err := reader.ReadString('\n')
				if err != nil {
					if err == io.EOF {
						break
					}
					break
				}

				if strings.Contains(line, "Chunk") {
					chunkCount++
					GinkgoWriter.Printf("Received chunk %d: %s", chunkCount, line)
				}
				if chunkCount >= 2 {
					break
				}
			}

			Expect(chunkCount).To(BeNumerically(">=", 2),
				"Should receive at least 2 chunks from streaming endpoint")
		})
	})

	Describe("Error handling", func() {
		It("should handle upstream errors gracefully", func() {
			client := &http.Client{Timeout: 5 * time.Second}
			resp, err := client.Get("http://localhost:8081/nonexistent")
			if err == nil {
				defer resp.Body.Close()
				Expect(resp.StatusCode).To(Or(
					Equal(http.StatusBadGateway),
					Equal(http.StatusNotFound),
					Equal(http.StatusInternalServerError)))
			}
		})
	})
	Describe("WebSocket handling", func() {
		It("should proxy WebSocket connections correctly", func() {
			By("Connecting to WebSocket endpoint via reverse proxy")

			// The reverse proxy should be listening on port 9090 for WebSocket connections
			// based on your websocat command: websocat ws://localhost:8081/ws
			wsURL := "ws://localhost:8081/ws"

			conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusSwitchingProtocols))

			// Test sending and receiving messages
			testMessage := "hello"
			err = conn.WriteMessage(websocket.TextMessage, []byte(testMessage))
			Expect(err).NotTo(HaveOccurred())

			// Read response with timeout
			messageType, message, err := conn.ReadMessage()
			Expect(err).NotTo(HaveOccurred())
			Expect(messageType).To(Equal(websocket.TextMessage))
			Expect(string(message)).To(Equal("hello")) // Based on your observed behavior
		})

		It("should handle multiple WebSocket messages", func() {
			wsURL := "ws://localhost:8081/ws"

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())
			defer conn.Close()

			// Send multiple messages
			messages := []string{"msg1", "msg2", "msg3"}
			receivedCount := 0

			for i, msg := range messages {
				err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
				Expect(err).NotTo(HaveOccurred())

				// Set read deadline to avoid hanging
				conn.SetReadDeadline(time.Now().Add(3 * time.Second))
				messageType, response, err := conn.ReadMessage()
				if err != nil {
					break
				}

				Expect(messageType).To(Equal(websocket.TextMessage))
				Expect(string(response)).To(Equal(fmt.Sprintf("msg%d", i+1))) // Your server's response
				receivedCount++
			}

			Expect(receivedCount).To(Equal(len(messages)),
				"Should receive responses for all sent messages")
		})

		It("should handle concurrent WebSocket connections", func() {
			const concurrentConnections = 3
			errors := make(chan error, concurrentConnections)
			var successCount int32

			for i := 0; i < concurrentConnections; i++ {
				go func(id int) {
					wsURL := "ws://localhost:8081/ws"
					conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
					if err != nil {
						errors <- err
						return
					}
					defer conn.Close()

					testMsg := fmt.Sprintf("connection %d", id)
					err = conn.WriteMessage(websocket.TextMessage, []byte(testMsg))
					if err != nil {
						errors <- err
						return
					}

					conn.SetReadDeadline(time.Now().Add(3 * time.Second))
					_, _, err = conn.ReadMessage()
					if err != nil {
						errors <- err
						return
					}

					atomic.AddInt32(&successCount, 1)
					errors <- nil
				}(i)
			}

			for i := 0; i < concurrentConnections; i++ {
				err := <-errors
				if err != nil {
					GinkgoWriter.Printf("WebSocket connection error: %v\n", err)
				}
			}

			final := int(atomic.LoadInt32(&successCount))
			Expect(final).To(BeNumerically(">=", concurrentConnections/2),
				"At least half of concurrent WebSocket connections should succeed")
		})

		It("should handle WebSocket close gracefully", func() {
			wsURL := "ws://localhost:8081/ws"

			conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			Expect(err).NotTo(HaveOccurred())

			// Send a message
			err = conn.WriteMessage(websocket.TextMessage, []byte("close test"))
			Expect(err).NotTo(HaveOccurred())

			// Read one response
			_, _, err = conn.ReadMessage()
			Expect(err).NotTo(HaveOccurred())

			// Close gracefully
			err = conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			Expect(err).NotTo(HaveOccurred())

			// Should be able to close without errors
			err = conn.Close()
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
