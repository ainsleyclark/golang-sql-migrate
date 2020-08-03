package migrate

import (
	"database/sql"
	"fmt"
	"github.com/gookit/color"
	"io/ioutil"
	"os"
	"sort"
	"time"
)

type Migrate struct {
	db            *sql.DB
	databaseName  string
	migrationPath string
	version       int
	verbose       bool
}

type migration struct {
	id          int    `db:"id"`
	name        string `db:"migration"`
	version     int    `db:"batch"`
	createdAt   string
	pathUp      string
	pathDown    string
	fail        bool
	hasMigrated bool
}

func newInstance() *Migrate {
	return &Migrate{}
}

// New migrate returns an instance of Migrate from an existing database source.
// Takes in a type of sql DB, database name and the migration path. You are,
// responsible for closing down the underlying source and database client
// if necessary.
func NewInstance(db *sql.DB, databaseName string, migrationPath string, verbose bool) (*Migrate, error) {
	m := newInstance()

	// Ping database.
	m.db = db
	if err := m.checkDatabaseConnection(); err != nil {
		return nil, err
	}

	// Check if database name exists in schema.
	if err := m.checkDatabaseExists(databaseName); err != nil {
		return nil, err
	}
	m.databaseName = databaseName

	// Process the migration path, strip trailing slash
	migrationPath, err := processDirectory(migrationPath)
	if err != nil {
		return nil, err
	}
	m.migrationPath = migrationPath

	//Create migration table if it does not exist.
	if m.tableExists() == false {
		if err := m.createTable(); err != nil {
			return nil, err
		}
	}

	// Set the current version number
	version, err := m.GetVersion()
	if err != nil {
		return nil, err
	}
	m.version = version

	// Set verbose
	m.verbose = verbose

	return m, nil
}

// Get the current batch number that is stored in the database.
// Will return 0 if no migrations have been run.
func (m *Migrate) GetVersion() (int, error) {
	rows, err := m.db.Query("SELECT MAX(batch) AS version FROM migrations")

	if err != nil {
		return 0, fmt.Errorf("cannot get the current database version - %w", err)
	} else {
		var version int
		for rows.Next() {
			if err := rows.Scan(&version); err != nil {
				return 0, err
			}
		}
		return version, nil
	}
}

// Up looks at the currently active migration batch
// number and will migrate all the way up
// (applying all up migrations).
func (m *Migrate) Up() error {

	migrations, err := m.getMigrateFiles()
	if err != nil {
		return err
	}

	isDirtyMigration := false
	for k, v := range migrations {

		if !v.hasMigrated {

			if m.verbose {
				color.Yellow.Print("Migrating:  ")
				fmt.Println(v.name)
			}

			if !fileExists(v.pathUp) {
				return fmt.Errorf("migration file down does not exist - %v", v.pathUp)
			}

			contents, err := getFileContents(v.pathUp)
			if err != nil {
				if m.verbose {
					color.Red.Print("Failure:    ")
					fmt.Println(v.name)
					color.Red.Println(err)
				}
				migrations[k].fail = true
				isDirtyMigration = true
				break
			}

			if err := m.run(contents); err != nil {
				if m.verbose {
					color.Red.Print("Failure:    ")
					fmt.Println(v.name)
					color.Red.Println(err)
				}
				migrations[k].fail = true
				isDirtyMigration = true
				break
			}

			if m.verbose {
				color.Green.Print("Migrated:   ")
				fmt.Println(v.name)
			}

			migrations[k].fail = false
		}
	}

	// Check if there was a dirty migration, if there was, run drop commands.
	// If there wasn't insert the migration into the database.
	if isDirtyMigration {

		if m.verbose {
			color.Green.Println("Rolling back...")
		}

		for _, v := range migrations {
			if !v.fail {
				contents, err := getFileContents(v.pathDown)
				if err != nil {
					color.Red.Println(err)
				}
				if err := m.run(contents); err != nil {
					color.Red.Println(err)
				}
			}
		}

		if m.verbose {
			color.Green.Println("Rolled back successfully")
		}

	} else {
		if err := m.create(migrations); err != nil {
			return err
		}
	}

	return nil
}

func (m *Migrate) Down() error {
	if _, err := m.db.Exec("DROP DATABASE " + m.databaseName + ";"); err != nil {
		return fmt.Errorf("cannot drop database %v - %w", m.databaseName, err)
	}

	return nil
}

// Rollback will get the latest version in the database and execute
// any files that are with the .down.sql extension.
func (m *Migrate) Rollback() error {

	// Get all the migration files.
	migrations, err := m.getMigrateFiles()
	if err != nil {
		return err
	}

	// Get the current migration version
	currentVersion, err := m.GetVersion()
	if err != nil {
		return err
	}

	hasRolledBack := true
	for _, v := range migrations {

		if v.version == currentVersion {

			if !fileExists(v.pathDown) {
				return fmt.Errorf("migration file down does not exist - %v", v.pathDown)
			}

			contents, err := getFileContents(v.pathDown)
			if err != nil {
				if m.verbose {
					color.Red.Print("Failure:    ")
					fmt.Println(v.name)
					color.Red.Println(err)
				}
				hasRolledBack = false
				break
			}

			if err := m.run(contents); err != nil {
				if m.verbose {
					color.Red.Print("Failure:    ")
					fmt.Println(v.name)
					color.Red.Println(err)
				}
				hasRolledBack = false
				break
			}

			if err := m.delete(v.name); err != nil {
				if m.verbose {
					color.Red.Print("Failure:    ")
					fmt.Println(v.name)
					color.Red.Println(err)
				}
				hasRolledBack = false
			}
		}
	}

	if !hasRolledBack {
		// TODO - What happens if it hasn't rolled back?
	} else {
		if m.verbose {
			color.Green.Println("Rolled back successfully")
		}
	}

	return nil
}

// Fresh will drop the whole database, create it and run all the
// pending migrations.
func (m *Migrate) Fresh() error {
	if err := m.DropAndCreate(); err != nil {
		return err
	}

	if err := m.Up(); err != nil {
		return err
	}

	return nil
}

// Drop will drop the whole database and create it again. Note it
// is not the same as fresh, as fresh that will run all the
// migrations over again.
func (m *Migrate) DropAndCreate() error {
	if err := m.Down(); err != nil {
		return err
	}

	if err := m.createDatabase(); err != nil {
		return err
	}

	if err := m.createTable(); err != nil {
		return err
	}

	if m.verbose {
		color.Green.Println("Database dropped & created successfully")
	}

	return nil
}

// Make will create up and down sql files based on the migration path
func (m *Migrate) Make(fileName string) error {
	filePath := m.migrationPath

	// TODO - Create stub files
	files := make(map[string]string)
	files["up.sql"] = "CREATE TABLE `tablename` ();"
	files["down.sql"] = "DROP TABLE `tablename`;"

	var err error
	for k, v := range files {
		var timeString string = time.Now().Format("2006_02_01_15_0405")
		sqlFileName := filePath + "/" + timeString + "_" + fileName + "." + k

		err = ioutil.WriteFile(sqlFileName, []byte(v), 0755)
	}

	if err != nil {
		return fmt.Errorf("unable to create migration file: %w", err)
	}

	return nil
}

// Get the migration files based on the kind args (up or down)
// Will return the migration (which holds the file name with no extensions),
// and the batch number.
func (m *Migrate) getMigrateFiles() (map[string]*migration, error) {

	// Get all migration files.
	files, err := getFiles(m.migrationPath)
	if err != nil {
		return nil, err
	}
	files = m.sort(files, true)

	// Get the current migration version
	version, err := m.GetVersion()
	if err != nil {
		return nil, err
	}

	// Build up migrations array.
	migrationsFound := make(map[string]*migration)
	for _, f := range files {
		fileName := f.Name()
		createdAt := fileName[0:18]
		name := m.getDBFileName(fileName)

		if _, err := migrationsFound[name]; !err {
			migrationsFound[name] = &migration{
				name:        name,
				version:     version + 1,
				createdAt:   createdAt,
				pathUp:      m.migrationPath + "/" + name + ".up.sql",
				pathDown:    m.migrationPath + "/" + name + ".down.sql",
				fail:        false,
				hasMigrated: false,
			}
		}
	}

	// Get current database migrations stored
	migrations, err := m.get(false)
	if err != nil {
		return nil, err
	}

	// Check if the migrations have already been run and assign version
	for _, v := range migrations {
		for k, m := range migrationsFound {
			if m.name == v.name {
				migrationsFound[k].version = v.version
				migrationsFound[k].hasMigrated = true
			}
		}
	}

	return migrationsFound, nil
}

// Sort will filter through the migration files passed depending
// on the up boolean passed.
func (m *Migrate) sort(files []os.FileInfo, up bool) []os.FileInfo {
	if up {
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() > files[j].Name()
		})
	} else {
		sort.Slice(files, func(i, j int) bool {
			return files[i].Name() < files[j].Name()
		})
	}

	return files
}

// Run the migration. Executes the sql provided.
func (m *Migrate) run(sql string) error {
	if _, err := m.db.Exec(sql); err != nil {
		return err
	}

	return nil
}
