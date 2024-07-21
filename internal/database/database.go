package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/mattn/go-sqlite3"
)

// Represents a service that interacts with a database.
type Service interface {
	// Returns a map of health status information. The keys and values in the map
	// are service-specific.
	Health() map[string]string

	// Terminates the database connection, returning an error if the connection
	// cannot be closed.
	Close() error

	// CreateEvent creates a new event entry in the database.
	// It returns the ID of the newly created event entry.
	// If an error occurs during the insertion, it returns the error.

	// Creates a new Event entry in the database. Returns the ID of the newly
	// created event entry or an error if the operation fails.
	CreateEvent(e EventEntry) (string, error)

	// Retrieves an Event entry from the DB with the given ID. Returns the Event
	// entry if found, or an error if the operation fails.
	GetEvent(id string) (EventEntry, error)

	// Retrieves all Event entries from the DB. Returns a slice of Event entries
	// if found, or an error if the operation fails.
	GetEvents() ([]EventEntry, error)
}

type service struct {
	db *sql.DB
}

var (
	dbUrl      = os.Getenv("DB_URL")
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	db, err := sql.Open("sqlite3", dbUrl)
	if err != nil {
		// This will not be a connection error, but a DSN parse error or
		// another initialization error.
		log.Fatal(err)
	}

	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf(fmt.Sprintf("db down: %v", err)) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", dbUrl)
	return s.db.Close()
}

// CreateEvent creates a new event entry in the database.
// It returns the ID of the newly created event entry.
// If an error occurs during the insertion, it returns the error.
func (s *service) CreateEvent(e EventEntry) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	newId := uuid.NewString()
	query := "INSERT INTO events (ID, Type, Data, Timestamp) VALUES (?, ?, ?, ?)"
	_, err := s.db.ExecContext(ctx, query, newId, e.Type, e.Data, e.Timestamp)
	if err != nil {
		return "", err
	}

	return newId, nil
}

// CreateEvents creates multiple event entries in the database.
// It returns the IDs of the newly created event entries.
// If an error occurs during the insertion, it returns the error.
func (s *service) CreateEvents(events []EventEntry) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	var ids []string
	query := "INSERT INTO events (ID, Type, Data, Timestamp) VALUES (?, ?, ?, ?)"
	for _, e := range events {
		newId := uuid.NewString()
		_, err := s.db.ExecContext(ctx, query, newId, e.Type, e.Data, e.Timestamp)
		if err != nil {
			return nil, err
		}
		ids = append(ids, newId)
	}

	return ids, nil
}

// GetEvent retrieves an event entry from the database based on the event ID.
// It returns the event entry if found, or an error if the retrieval fails.
func (s *service) GetEvent(id string) (EventEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	query := "SELECT id, type, data, timestamp FROM events WHERE id = ?"
	row := s.db.QueryRowContext(ctx, query, id)

	var event EventEntry
	err := row.Scan(&event.ID, &event.Type, &event.Data, &event.Timestamp)
	if err != nil {
		return EventEntry{}, err
	}

	return event, nil
}

// GetEvents retrieves all event entries from the database.
// It returns a slice of event entries if found, or an error if the retrieval fails.
func (s *service) GetEvents() ([]EventEntry, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	query := "SELECT id, type, data, timestamp FROM events"
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []EventEntry
	for rows.Next() {
		var e EventEntry
		err := rows.Scan(&e.ID, &e.Type, &e.Data, &e.Timestamp)
		if err != nil {
			return nil, err
		}
		events = append(events, e)
	}

	return events, nil
}
