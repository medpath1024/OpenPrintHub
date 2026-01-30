package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestNewWebSocketHub(t *testing.T) {
	hub := NewWebSocketHub()
	if hub == nil {
		t.Fatal("NewWebSocketHub returned nil")
	}
	if hub.clients == nil {
		t.Error("clients map is nil")
	}
	if hub.broadcast == nil {
		t.Error("broadcast channel is nil")
	}
	if hub.register == nil {
		t.Error("register channel is nil")
	}
	if hub.unregister == nil {
		t.Error("unregister channel is nil")
	}
}

func TestWebSocketHub_ClientCount_Empty(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	count := hub.ClientCount()
	if count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}
}

func TestWebSocketHub_Broadcast(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Test broadcasting with no clients (should not block)
	msg := WebSocketMessage{
		Type: "test",
		Data: map[string]string{"key": "value"},
	}

	// Should complete without blocking
	done := make(chan bool)
	go func() {
		hub.Broadcast(msg)
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Broadcast blocked with no clients")
	}
}

func TestWebSocketMessage_Marshal(t *testing.T) {
	msg := WebSocketMessage{
		Type: "job_status",
		Data: map[string]interface{}{
			"job_id": "123",
			"status": "completed",
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed WebSocketMessage
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.Type != "job_status" {
		t.Errorf("Expected type 'job_status', got '%s'", parsed.Type)
	}
}

func TestWebSocketHub_RegisterUnregister(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Allow hub to start
	time.Sleep(50 * time.Millisecond)

	// Create a mock client
	client := &WebSocketClient{
		hub:  hub,
		send: make(chan []byte, 256),
	}

	// Register
	hub.register <- client
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client after register, got %d", hub.ClientCount())
	}

	// Unregister
	hub.unregister <- client
	time.Sleep(50 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after unregister, got %d", hub.ClientCount())
	}
}

func TestWebSocketHub_BroadcastToClients(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Allow hub to start
	time.Sleep(50 * time.Millisecond)

	// Create mock clients
	client1 := &WebSocketClient{
		hub:  hub,
		send: make(chan []byte, 256),
	}
	client2 := &WebSocketClient{
		hub:  hub,
		send: make(chan []byte, 256),
	}

	// Register clients
	hub.register <- client1
	hub.register <- client2
	time.Sleep(50 * time.Millisecond)

	// Broadcast a message
	msg := WebSocketMessage{
		Type: "test",
		Data: "hello",
	}
	hub.Broadcast(msg)

	// Allow broadcast to propagate
	time.Sleep(50 * time.Millisecond)

	// Check both clients received the message
	select {
	case <-client1.send:
		// Good
	case <-time.After(time.Second):
		t.Error("Client 1 did not receive message")
	}

	select {
	case <-client2.send:
		// Good
	case <-time.After(time.Second):
		t.Error("Client 2 did not receive message")
	}
}

func TestWebSocketHub_MultipleClients(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	time.Sleep(50 * time.Millisecond)

	numClients := 10
	clients := make([]*WebSocketClient, numClients)

	// Register multiple clients
	for i := 0; i < numClients; i++ {
		clients[i] = &WebSocketClient{
			hub:  hub,
			send: make(chan []byte, 256),
		}
		hub.register <- clients[i]
	}

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != numClients {
		t.Errorf("Expected %d clients, got %d", numClients, hub.ClientCount())
	}

	// Unregister all
	for i := 0; i < numClients; i++ {
		hub.unregister <- clients[i]
	}

	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients, got %d", hub.ClientCount())
	}
}

func TestWebSocketHub_ConcurrentOperations(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	time.Sleep(50 * time.Millisecond)

	var wg sync.WaitGroup
	numOps := 50

	// Concurrent register/unregister/broadcast
	for i := 0; i < numOps; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := &WebSocketClient{
				hub:  hub,
				send: make(chan []byte, 256),
			}
			hub.register <- client

			// Broadcast
			hub.Broadcast(WebSocketMessage{Type: "test", Data: "data"})

			hub.unregister <- client
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	// All clients should be unregistered
	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after concurrent operations, got %d", hub.ClientCount())
	}
}

// Integration test with actual WebSocket connection
func TestWebSocket_Integration(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Create a test server
	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	// Connect a WebSocket client
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	// Allow connection to be registered
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", hub.ClientCount())
	}
}

func TestWebSocket_PingPong(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	// Send a ping message
	pingMsg := map[string]string{"type": "ping"}
	if err := conn.WriteJSON(pingMsg); err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Read response
	var response map[string]interface{}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response["type"] != "pong" {
		t.Errorf("Expected 'pong' response, got '%v'", response["type"])
	}
}

func TestWebSocket_Subscribe(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	// Send a subscribe message
	subMsg := map[string]string{"type": "subscribe"}
	if err := conn.WriteJSON(subMsg); err != nil {
		t.Fatalf("Failed to send subscribe: %v", err)
	}

	// Read response
	var response map[string]interface{}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if response["type"] != "subscribed" {
		t.Errorf("Expected 'subscribed' response, got '%v'", response["type"])
	}
}

func TestWebSocket_ReceiveBroadcast(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	// Allow connection to be registered
	time.Sleep(100 * time.Millisecond)

	// Broadcast a message
	hub.Broadcast(WebSocketMessage{
		Type: "job_status",
		Data: map[string]string{"job_id": "test-123", "status": "completed"},
	})

	// Read the broadcast
	var response map[string]interface{}
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("Failed to read broadcast: %v", err)
	}

	if response["type"] != "job_status" {
		t.Errorf("Expected type 'job_status', got '%v'", response["type"])
	}
}

func TestWebSocket_MultipleConnections(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	numConnections := 5
	connections := make([]*websocket.Conn, numConnections)

	// Connect multiple clients
	for i := 0; i < numConnections; i++ {
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		connections[i] = conn
	}
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()

	// Allow connections to be registered
	time.Sleep(200 * time.Millisecond)

	if hub.ClientCount() != numConnections {
		t.Errorf("Expected %d clients, got %d", numConnections, hub.ClientCount())
	}

	// Broadcast a message
	hub.Broadcast(WebSocketMessage{
		Type: "test",
		Data: "broadcast to all",
	})

	// All clients should receive the message
	for i, conn := range connections {
		var response map[string]interface{}
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		if err := conn.ReadJSON(&response); err != nil {
			t.Errorf("Client %d failed to read broadcast: %v", i, err)
		}
	}
}

func TestWebSocket_ConnectionClose(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// Allow connection to be registered
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Errorf("Expected 1 client before close, got %d", hub.ClientCount())
	}

	// Close the connection
	conn.Close()

	// Allow unregister to happen
	time.Sleep(200 * time.Millisecond)

	if hub.ClientCount() != 0 {
		t.Errorf("Expected 0 clients after close, got %d", hub.ClientCount())
	}
}

func TestWebSocket_UnknownMessageType(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	// Send an unknown message type
	unknownMsg := map[string]string{"type": "unknown_type"}
	if err := conn.WriteJSON(unknownMsg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Should not crash or disconnect
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Error("Client should still be connected after unknown message")
	}
}

func TestWebSocket_InvalidJSON(t *testing.T) {
	hub := NewWebSocketHub()
	go hub.Run()

	router := gin.New()
	router.GET("/ws", func(c *gin.Context) {
		hub.HandleConnection(c.Writer, c.Request)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"

	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	defer conn.Close()

	// Send invalid JSON
	if err := conn.WriteMessage(websocket.TextMessage, []byte("invalid json")); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Should handle gracefully
	time.Sleep(100 * time.Millisecond)

	if hub.ClientCount() != 1 {
		t.Error("Client should still be connected after invalid JSON")
	}
}

func BenchmarkWebSocketHub_Broadcast(b *testing.B) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Register some mock clients
	for i := 0; i < 100; i++ {
		client := &WebSocketClient{
			hub:  hub,
			send: make(chan []byte, 256),
		}
		hub.register <- client
		// Drain the send channel
		go func(c *WebSocketClient) {
			for range c.send {
			}
		}(client)
	}

	time.Sleep(100 * time.Millisecond)

	msg := WebSocketMessage{
		Type: "test",
		Data: map[string]string{"key": "value"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.Broadcast(msg)
	}
}

func BenchmarkWebSocketHub_ClientCount(b *testing.B) {
	hub := NewWebSocketHub()
	go hub.Run()

	// Register some clients
	for i := 0; i < 100; i++ {
		client := &WebSocketClient{
			hub:  hub,
			send: make(chan []byte, 256),
		}
		hub.register <- client
	}

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hub.ClientCount()
	}
}
