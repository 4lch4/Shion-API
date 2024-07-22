package server

import (
	"fmt"
	"net/http"
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

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	// All routes are to be prefixed with /api/v1, e.g. /api/v1/events.
	rootGroup := r.Group("/api/v1")

	// All WebSocket routes are to be prefixed with /ws, e.g. /api/v1/ws/events.
	wsGroup := rootGroup.Group("/ws")

	rootGroup.GET("/health/db", s.dbHealthHandler)
	rootGroup.GET("/health/liveness", s.basicHealthHandler)
	rootGroup.GET("/health/readiness", s.basicHealthHandler)

	rootGroup.GET("/event", s.getEventHandler)
	rootGroup.POST("/event", s.incomingEventHandler)

	rootGroup.GET("/events", s.getEventsHandler)
	rootGroup.POST("/events", s.incomingEventsHandler)

	wsGroup.GET("/events", s.wsEventHandler)

	return r
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

// Retrieves a single event by its ID. Returns the event if found, or an error
// if the operation fails.
func (s *Server) getEventHandler(c *gin.Context) {
	eventId := c.Param("id")

	event, err := s.db.GetEventByID(eventId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

// Retrieves the latest events up to a maximum number of events. Returns a slice
// of event entries if found, or an error if the operation fails.
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

// Handles an incoming Event entry. Returns the event if successful, or an error
// if the operation fails.
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
		EventEntry: insertedEvent,
	}

	c.JSON(http.StatusOK, resp)
}

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
			EventEntry: insertedEvent,
		})
	}

	c.JSON(http.StatusOK, responses)
}

func (s *Server) dbHealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

func (s *Server) basicHealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
