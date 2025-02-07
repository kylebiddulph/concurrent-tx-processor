package transaction

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"os"
	"sort"
	"sync"
	"testing"
	"time"
)

func createConnection(t *testing.T) *sql.DB {
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/%s",
		os.Getenv("MYSQL_USER"),
		os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("MYSQL_DATABASE")))
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		t.Fatalf("Could not ping database: %v", err)
	}

	return db
}

func TestDatabaseConnection(t *testing.T) {
	// Try to create the connection
	db := createConnection(t)
	defer db.Close()

	// If we get here, basic connection worked (due to Ping in createConnection)
	// Let's try a simple query to really verify
	var result int
	err := db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Fatalf("Failed to execute test query: %v", err)
	}

	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}

	t.Log("Successfully connected to database and executed test query")
}

func TestReadThenUpdateRace(t *testing.T) {
	db := createConnection(t)
	defer db.Close()

	iterations := 10
	raceConditionsDetected := 0

	for iter := 0; iter < iterations; iter++ {
		t.Logf("\n=== Iteration %d ===", iter+1)

		// Create initial transaction in PENDING state
		txID := fmt.Sprintf("tx_concurrent_test_%d", iter)
		err := CreateTransaction(db, txID, 100.00, "PENDING")
		if err != nil {
			t.Fatalf("Failed to create transaction: %v", err)
		}

		// Channel to track the sequence of events
		type systemEvent struct {
			system    string
			action    string // "read" or "update"
			status    string
			timestamp time.Time
		}
		eventChan := make(chan systemEvent, 4) // 2 reads + 2 updates
		var wg sync.WaitGroup

		// Simulate System A checking then updating
		wg.Add(1)
		go func() {
			defer wg.Done()

			// First read the status
			var currentStatus string
			err := db.QueryRow("SELECT status FROM transactions WHERE transaction_id = ?", txID).Scan(&currentStatus)
			readTime := time.Now()
			if err != nil {
				t.Logf("System A failed to read status: %v", err)
				return
			}

			eventChan <- systemEvent{
				system:    "System A",
				action:    "read",
				status:    currentStatus,
				timestamp: readTime,
			}

			// If status is PENDING, try to update
			if currentStatus == "PENDING" {
				err = UpdateStatus(db, txID, "COMPLETED")
				updateTime := time.Now()
				eventChan <- systemEvent{
					system:    "System A",
					action:    "update",
					status:    "COMPLETED",
					timestamp: updateTime,
					// success implicitly shown by lack of error
				}
			}
		}()

		// Simulate System B checking then updating
		wg.Add(1)
		go func() {
			defer wg.Done()

			// First read the status
			var currentStatus string
			err := db.QueryRow("SELECT status FROM transactions WHERE transaction_id = ?", txID).Scan(&currentStatus)
			readTime := time.Now()
			if err != nil {
				t.Logf("System B failed to read status: %v", err)
				return
			}

			eventChan <- systemEvent{
				system:    "System B",
				action:    "read",
				status:    currentStatus,
				timestamp: readTime,
			}

			// If status is PENDING, try to update
			if currentStatus == "PENDING" {
				err = UpdateStatus(db, txID, "COMPLETED")
				updateTime := time.Now()
				eventChan <- systemEvent{
					system:    "System B",
					action:    "update",
					status:    "COMPLETED",
					timestamp: updateTime,
					// success implicitly shown by lack of error
				}
			}
		}()

		wg.Wait()
		close(eventChan)

		// Collect all events
		var events []systemEvent
		for event := range eventChan {
			events = append(events, event)
		}

		// Sort events by timestamp
		sort.Slice(events, func(i, j int) bool {
			return events[i].timestamp.Before(events[j].timestamp)
		})

		// Count successful updates
		updateCount := 0
		for _, event := range events {
			if event.action == "update" {
				updateCount++
			}
		}

		// Log the sequence of events
		t.Log("\nEvents in chronological order:")
		for _, event := range events {
			t.Logf("%s: %s (status: %s) at %v",
				event.system,
				event.action,
				event.status,
				event.timestamp.Format("15:04:05.000000"))
		}

		// Check if both systems saw PENDING and then both updated
		var sawPendingA, sawPendingB bool
		var updatedA, updatedB bool

		for _, event := range events {
			if event.action == "read" && event.status == "PENDING" {
				if event.system == "System A" {
					sawPendingA = true
				} else {
					sawPendingB = true
				}
			}
			if event.action == "update" {
				if event.system == "System A" {
					updatedA = true
				} else {
					updatedB = true
				}
			}
		}

		if sawPendingA && sawPendingB && updatedA && updatedB {
			raceConditionsDetected++
			t.Errorf("Race condition detected: Both systems saw PENDING status and performed updates")
		}

		// Verify final state
		var finalStatus string
		err = db.QueryRow("SELECT status FROM transactions WHERE transaction_id = ?", txID).Scan(&finalStatus)
		if err != nil {
			t.Fatalf("Failed to get final status: %v", err)
		}
		t.Logf("Final transaction status: %s", finalStatus)
	}

	// Final summary
	t.Logf("\n=== Test Summary ===")
	t.Logf("Total iterations: %d", iterations)
	t.Logf("Race conditions detected: %d", raceConditionsDetected)
	if raceConditionsDetected > 0 {
		t.Errorf("Race condition detected in %d out of %d iterations", raceConditionsDetected, iterations)
	}
}
