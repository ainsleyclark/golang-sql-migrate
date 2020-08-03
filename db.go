package migrate

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
)

// Check database to see if it is valid. Ping database to check connection.
func (m *Migrate) checkDatabaseConnection() error {
	if err := m.db.Ping(); err != nil {
		return fmt.Errorf("could not ping the database - %w", err)
	}

	return nil
}

// Check if database exists based on the database name argument
func (m *Migrate) checkDatabaseExists(databaseName string) error {
	_, err := m.db.Exec("SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?", databaseName)

	if err != nil {
		return fmt.Errorf("database not found - %s", databaseName)
	}

	return nil
}

// Create the new migration table, with the column values of ID,
// Migration (which holds the file name with no extensions),
// and the batch number.
func (m *Migrate) createTable() error {
	_, err := m.db.Exec("CREATE TABLE migrations (`id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY, `migration` VARCHAR(255), `batch` INT)")

	if err != nil {
		return fmt.Errorf("could not create the migration table - %w", err)
	}

	return nil
}

func (m *Migrate) createDatabase() error {
	if _, err := m.db.Exec("CREATE DATABASE " + m.databaseName + "; USE " + m.databaseName + ";"); err != nil {
		return fmt.Errorf("cannot create database %v - %w", m.databaseName, err)
	}

	return nil
}

// Check to see if the migration table exists, returns bool dependant
// on result.
func (m *Migrate) tableExists() bool {
	var q string
	err := m.db.QueryRow("(SELECT * FROM information_schema.tables WHERE table_schema = ? AND table_name = 'migrations' LIMIT 1);", m.databaseName).Scan(&q)

	if err == sql.ErrNoRows {
		return false
	}

	return true
}

// Gets all the migration files currently stored within the
// database.
func (m *Migrate) get(currentVersion bool) ([]migration, error) {

	// Get the current migration version
	version, err := m.GetVersion()
	if err != nil {
		return nil, err
	}

	var q string = "SELECT * FROM migrations"
	if currentVersion {
		q = "SELECT * FROM migrations WHERE batch = '" + strconv.Itoa(version) + "'"
	}

	rows, err := m.db.Query(q)
	if err != nil {
		return nil, err
	}

	var migrations []migration
	for rows.Next() {
		var m migration
		err = rows.Scan(&m.id, &m.name, &m.version)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, m)
	}

	return migrations, nil
}

// Add migrate files and batch to database when going up.
// This will create new records in the database with
// the batch number and associative file.
func (m *Migrate) create(migrations map[string]*migration) error {
	for _, v := range migrations {
		if !v.hasMigrated {
			_, err := m.db.Exec("INSERT INTO migrations (migration, batch) VALUES (?, ?);", v.name, v.version)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Delete migrate files and batch to database when going down.
// This will delete records in the database with
// the batch number and associative file.
func (m *Migrate) delete(name string) error {
	_, err := m.db.Exec("DELETE FROM migrations WHERE migration = '" + name + "'")

	if err != nil {
		return err
	}

	return nil
}

// Get the database file name as stored, removes extensions and up/down.
func (m *Migrate) getDBFileName(fileName string) string {
	fileName = strings.Replace(fileName, ".up.sql", "", -1)
	fileName = strings.Replace(fileName, ".down.sql", "", -1)

	return fileName
}
