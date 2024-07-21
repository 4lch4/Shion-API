package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/4lch4/shion-api/internal/database"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type EventResponse struct {
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	InsertId  string `json:"insertId"`
}

var wsUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (s *Server) RegisterRoutes() http.Handler {
	r := gin.Default()

	rootGroup := r.Group("/api/v1")
	wsGroup := rootGroup.Group("/ws")

	// rootGroup.GET("/", s.HelloWorldHandler)

	rootGroup.GET("/health/db", s.dbHealthHandler)
	rootGroup.GET("/health/liveness", s.basicHealthHandler)
	rootGroup.GET("/health/readiness", s.basicHealthHandler)

	rootGroup.GET("/event", s.getEventHandler)
	rootGroup.POST("/event", s.incomingEventHandler)

	rootGroup.GET("/events", s.getEventsHandler)
	rootGroup.POST("/events", s.incomingEventsHandler)

	wsGroup.GET("/events", func(c *gin.Context) {
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
	})

	return r
}

func (s *Server) getEventHandler(c *gin.Context) {
	eventId := c.Param("id")

	event, err := s.db.GetEvent(eventId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, event)
}

func (s *Server) getEventsHandler(c *gin.Context) {
	events, err := s.db.GetEvents()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, events)
}

func (s *Server) incomingEventHandler(c *gin.Context) {
	var payload database.EventEntry

	if err := c.ShouldBind(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	insertId, err := s.db.CreateEvent(payload)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := EventResponse{
		Message:   fmt.Sprintf("Event received: %s - %s", payload.Type, payload.Data),
		Timestamp: payload.Timestamp,
		InsertId:  insertId,
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
		insertId, err := s.db.CreateEvent(entry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		responses = append(responses, EventResponse{
			Message:   fmt.Sprintf("Event received: %s - %s", entry.Type, entry.Data),
			Timestamp: entry.Timestamp,
			InsertId:  insertId,
		},
		)
	}

	c.JSON(http.StatusOK, responses)
}

func (s *Server) HelloWorldHandler(c *gin.Context) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	c.JSON(http.StatusOK, resp)
}

func (s *Server) dbHealthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, s.db.Health())
}

func (s *Server) basicHealthHandler(c *gin.Context) {
	c.String(http.StatusOK, "OK")
}
