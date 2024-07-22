package server

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/4lch4/shion-api/internal/database"

	_ "github.com/joho/godotenv/autoload"
)

type Server struct {
	port int

	db database.TursoDB
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("API_PORT"))
	NewServer := &Server{
		port: port,

		db: database.New(),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
