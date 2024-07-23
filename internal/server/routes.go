package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/4lch4/shion-api/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type EventResponse struct {
	// An optional message to be sent back to the client.
	Message string `json:"message"`

	// The event entry/entries that were created/queried/etc.
	EventEntry []database.EventEntry `json:"event_entry"`
}

var (
	// The username to be used for basic authentication.
	apiUsername = os.Getenv("API_USERNAME")

	// The password to be used for basic authentication.
	apiPassword = os.Getenv("API_PASSWORD")

	// Upgrader is used to upgrade an HTTP connection to a WebSocket connection.
	wsUpgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
)

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// All routes are to be prefixed with /api/v1, e.g. /api/v1/event.
	rootGroup := r.Group("/api/v1")

	// Apply the basicAuthMiddleware to all routes registered under the rootGroup.
	rootGroup.Use(basicAuthMiddleware())

	// All WebSocket routes are to be prefixed with /ws, e.g. /api/v1/ws/events.
	wsGroup := rootGroup.Group("/ws")

	rootGroup.GET("/health/db", s.dbHealthHandler)
	rootGroup.GET("/health/liveness", basicHealthHandler)
	rootGroup.GET("/health/readiness", basicHealthHandler)

	rootGroup.GET("/event", s.getEventHandler)
	rootGroup.POST("/event", s.incomingEventHandler)

	rootGroup.GET("/events", s.getEventsHandler)
	rootGroup.POST("/events", s.incomingEventsHandler)

	wsGroup.GET("/events", s.wsEventHandler)

	return r
}

// A basic auth middleware function I got from Phind made for Gin. It checks the
// request's basic auth credentials against the username and password provided
// in the environment variables. If the credentials are correct, the request is
// allowed to continue. If the credentials are incorrect, the request is aborted
// and a 401 Unauthorized response is sent back to the client.
func basicAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, pass, hasAuth := c.Request.BasicAuth()
		if !hasAuth || user != apiUsername || pass != apiPassword {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "Unauthorized"})
			return
		}
		c.Next()
	}
}

// A simple WebSocket handler that sends a message every second for testing.
// In the end, this endpoint will function similar to the createEvent endpoint,
// but will be able to handle a WebSocket connection for faster, more efficient
// communication.
func (s *Server) wsEventHandler(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("err:", err)
		return
	}
	defer conn.Close()
	for {
		conn.WriteMessage(websocket.TextMessage, []byte("Hello, WebSocket!"))
		time.Sleep(time.Second)
	}
}

// Handles requests to the GET /event/:id endpoint, which accepts a single event
// ID and returns the event with that ID, or an error if the operation fails.
func (s *Server) getEventHandler(c *gin.Context) {
	eventId := c.Param("id")

	event, err := s.db.GetEventByID(eventId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

// Handles requests to the GET /events endpoint, which accepts a query parameter
// for the maximum number of events to return. Returns a slice of the latest
// events up to the maximum number specified, or an error if the operation fails.
func (s *Server) getEventsHandler(c *gin.Context) {
	maxStr := c.DefaultQuery("max", "50")
	if maxStr == "" {
		maxStr = "50"
	}

	max, err := strconv.Atoi(maxStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	events, err := s.db.GetLatestEvents(max)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(events) == 0 {
		c.JSON(http.StatusOK, []database.EventEntry{})
		return
	}

	c.JSON(http.StatusOK, events)
}

// Handles requests to the POST /event endpoint, which accepts a single Event
// entry and inserts it into the database. Returns the event that was created if
// successful, or an error if the operation fails.
func (s *Server) incomingEventHandler(c *gin.Context) {
	var payload database.EventEntry

	if err := c.ShouldBind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	insertedEvent, err := s.db.CreateEvent(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := EventResponse{
		Message:    "Event successfully received!",
		EventEntry: []database.EventEntry{insertedEvent},
	}

	c.JSON(http.StatusOK, resp)
}

// Handles requests to the POST /events endpoint, which accepts an array of
// Event entries and inserts them into the database. Returns a slice of the
// events that were created if successful, or an error if the operation fails.
func (s *Server) incomingEventsHandler(c *gin.Context) {
	var entries []database.EventEntry
	var responses []EventResponse

	if err := c.ShouldBind(&entries); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, entry := range entries {
		insertedEvent, err := s.db.CreateEvent(entry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses = append(responses, EventResponse{
			Message:    "Event(s) successfully received!",
			EventEntry: []database.EventEntry{insertedEvent},
		})
	}

	c.JSON(http.StatusOK, responses)
}

func (s *Server) dbHealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

func basicHealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}

// A custom logger that outputs the client's IP address, the status code of the
// response, the latency of the request, the request method, and the request
// path. I've commented this out because I've realized this log message is
// output as well as the default debug messages so it's causing some things to
// be logged twice and very ugly. For example, the following is what appears
// when I send a request to the /api/v1/health/db endpoint:
// 2024/07/23 05:31:03 [GIN] 172.17.0.1 - 200 - 46.588µs - GET - /api/v1/health/db
// [GIN] 2024/07/23 - 05:31:03 | 200 |     212.452µs |      172.17.0.1 | GET      "/api/v1/health/db"
// func CustomLogger() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		startTime := time.Now()
//
// 		// Process request
// 		c.Next()
//
// 		// Calculate latency
// 		latency := time.Since(startTime)
//
// 		// Get client's IP address
// 		clientIP := c.ClientIP()
//
// 		// Log format: IP - StatusCode - Latency - Request Method - Request Path
// 		log.Printf("[GIN] %s - %d - %s - %s - %s",
// 			clientIP,
// 			c.Writer.Status(),
// 			latency,
// 			c.Request.Method,
// 			c.Request.URL.Path,
// 		)
// 	}
// }
