package database
import (
	"database/sql"
	"fmt"
	"app/Database/schema"
	configs "app/configs"
	"sync"

	_ "github.com/lib/pq"
)

var dbInstance *sql.DB
var dbInstanceError error
var dbOnce sync.Once

func GetPostgresDB(config *configs.Config) (*sql.DB, error) {
	dbOnce.Do(func() {
		connectionStr := config.GetDatabaseURL()
		db, err := sql.Open("postgres", connectionStr)
		if err != nil {
			dbInstanceError = fmt.Errorf("failed to connect to PostgreSQL: %v", err)
			return
		}
		err = db.Ping()
		if err != nil {
			dbInstanceError = fmt.Errorf("failed to ping PostgreSQL: %v", err)
			return
		}
		dbInstance = db
		// Create all user-related tables
		if err := schema.CreateAllTables(db); err != nil {
			dbInstanceError = fmt.Errorf("failed to create user tables: %v", err)
			dbInstance = nil
			return
		}

		// Create all photo-related tables
		if err := schema.CreateAllPhotoTables(db); err != nil {
			dbInstanceError = fmt.Errorf("failed to create photo tables: %v", err)
			dbInstance = nil
			return
		}

	})
	return dbInstance, dbInstanceError
}