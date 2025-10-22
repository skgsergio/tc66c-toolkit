package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/skgsergio/tc66-toolkit/lib/tc66c"
	"github.com/spf13/cobra"
	"go.bug.st/serial/enumerator"
)

//go:embed webui
var webuiFS embed.FS

var (
	webPortFlag string
	webAddrFlag string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start web server with UI for device interaction",
	Long: `Start a web server that serves a UI for interacting with TC66C devices.
The UI provides real-time monitoring via WebSocket connection.`,
	Run: func(cmd *cobra.Command, args []string) {
		executeWeb(webAddrFlag, webPortFlag)
	},
}

func init() {
	webCmd.Flags().StringVarP(&webAddrFlag, "address", "a", "localhost", "Address to bind the web server")
	webCmd.Flags().StringVarP(&webPortFlag, "web-port", "w", "8080", "Port for the web server")
	rootCmd.AddCommand(webCmd)
}

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local development
	},
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// WSResponse represents a WebSocket response
type WSResponse struct {
	Command string      `json:"command"`
	Success bool        `json:"success"`
	Error   string      `json:"error,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// SerialPortInfo contains information about a serial port from the OS
type SerialPortInfo struct {
	Name         string `json:"name"`
	IsUSB        bool   `json:"is_usb"`
	VID          string `json:"vid,omitempty"`
	PID          string `json:"pid,omitempty"`
	SerialNumber string `json:"serial_number,omitempty"`
}

// PollRequest represents the data for a poll command
type PollRequest struct {
	Port     string `json:"port"`
	Interval int    `json:"interval"` // interval in milliseconds
}

// Client represents a WebSocket client connection
type Client struct {
	conn          *websocket.Conn
	device        *tc66c.TC66C
	pollTicker    *time.Ticker
	pollStop      chan bool
	mu            sync.Mutex
	isPolling     bool
}

func executeWeb(addr, port string) {
	// Serve static files from embedded webui directory
	staticFS, err := fs.Sub(webuiFS, "webui")
	if err != nil {
		log.Fatalf("Failed to access webui directory: %v", err)
	}

	http.Handle("/", http.FileServer(http.FS(staticFS)))
	http.HandleFunc("/ws", handleWebSocket)

	listenAddr := fmt.Sprintf("%s:%s", addr, port)
	fmt.Printf("Starting web server on http://%s\n", listenAddr)
	fmt.Printf("Press Ctrl+C to stop the server\n")

	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("Failed to start web server: %v", err)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		conn:     conn,
		pollStop: make(chan bool),
	}

	defer func() {
		client.cleanup()
		conn.Close()
	}()

	log.Printf("WebSocket client connected from %s", r.RemoteAddr)

	for {
		var msg WSMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		client.handleMessage(msg)
	}

	log.Printf("WebSocket client disconnected from %s", r.RemoteAddr)
}

func (c *Client) handleMessage(msg WSMessage) {
	switch msg.Command {
	case "list-serial":
		c.handleListSerial()
	case "poll":
		c.handlePoll(msg.Data)
	case "stop":
		c.handleStop()
	case "close":
		c.handleClose()
	default:
		c.sendResponse(WSResponse{
			Command: msg.Command,
			Success: false,
			Error:   fmt.Sprintf("unknown command: %s", msg.Command),
		})
	}
}

func (c *Client) handleListSerial() {
	ports, err := enumerator.GetDetailedPortsList()
	if err != nil {
		c.sendResponse(WSResponse{
			Command: "list-serial",
			Success: false,
			Error:   fmt.Sprintf("failed to list serial ports: %v", err),
		})
		return
	}

	portInfos := make([]SerialPortInfo, 0, len(ports))

	for _, port := range ports {
		portInfo := SerialPortInfo{
			Name:         port.Name,
			IsUSB:        port.IsUSB,
			VID:          port.VID,
			PID:          port.PID,
			SerialNumber: port.SerialNumber,
		}

		portInfos = append(portInfos, portInfo)
	}

	c.sendResponse(WSResponse{
		Command: "list-serial",
		Success: true,
		Data:    portInfos,
	})
}

func (c *Client) handlePoll(data json.RawMessage) {
	var req PollRequest
	if err := json.Unmarshal(data, &req); err != nil {
		c.sendResponse(WSResponse{
			Command: "poll",
			Success: false,
			Error:   fmt.Sprintf("invalid poll request: %v", err),
		})
		return
	}

	// Validate interval
	if req.Interval < 100 {
		req.Interval = 100 // minimum 100ms
	}

	// Stop existing polling if any
	c.stopPolling()

	// Connect to device
	device, err := tc66c.NewTC66C(req.Port)
	if err != nil {
		c.sendResponse(WSResponse{
			Command: "poll",
			Success: false,
			Error:   fmt.Sprintf("failed to connect to device: %v", err),
		})
		return
	}

	// Validate device is in firmware mode
	if device.Mode != tc66c.ModeFirmware {
		device.Close()
		c.sendResponse(WSResponse{
			Command: "poll",
			Success: false,
			Error:   fmt.Sprintf("device must be in firmware mode (current mode: %s)", device.Mode),
		})
		return
	}

	c.mu.Lock()
	c.device = device
	c.isPolling = true
	c.mu.Unlock()

	// Send success response
	c.sendResponse(WSResponse{
		Command: "poll",
		Success: true,
		Data:    map[string]interface{}{"port": req.Port, "interval": req.Interval},
	})

	// Start polling in a goroutine
	go c.pollDevice(time.Duration(req.Interval) * time.Millisecond)
}

func (c *Client) pollDevice(interval time.Duration) {
	c.pollTicker = time.NewTicker(interval)
	defer c.pollTicker.Stop()

	for {
		select {
		case <-c.pollStop:
			return
		case <-c.pollTicker.C:
			c.mu.Lock()
			if c.device == nil {
				c.mu.Unlock()
				return
			}

			reading, err := c.device.GetReading()
			c.mu.Unlock()

			if err != nil {
				c.sendResponse(WSResponse{
					Command: "poll-data",
					Success: false,
					Error:   fmt.Sprintf("failed to get reading: %v", err),
				})
				continue
			}

			c.sendResponse(WSResponse{
				Command: "poll-data",
				Success: true,
				Data:    reading,
			})
		}
	}
}

func (c *Client) handleStop() {
	c.stopPolling()

	c.sendResponse(WSResponse{
		Command: "stop",
		Success: true,
	})
}

func (c *Client) handleClose() {
	c.sendResponse(WSResponse{
		Command: "close",
		Success: true,
	})

	// Close the WebSocket connection
	c.conn.Close()
}

func (c *Client) stopPolling() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isPolling {
		c.pollStop <- true
		c.isPolling = false
	}

	if c.device != nil {
		c.device.Close()
		c.device = nil
	}

	if c.pollTicker != nil {
		c.pollTicker.Stop()
		c.pollTicker = nil
	}
}

func (c *Client) cleanup() {
	c.stopPolling()
}

func (c *Client) sendResponse(resp WSResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.conn.WriteJSON(resp); err != nil {
		log.Printf("Failed to send WebSocket response: %v", err)
	}
}
