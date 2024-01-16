package inventoryengine

import (
	"database/sql"
	"fmt"

	// "github.com/aws/smithy-go/logging"
	_ "github.com/mattn/go-sqlite3"

	// "log"
	"errors"
	"time"
)

type DB struct {
	dbFilename string
	db *sql.DB
}

func NewDB() *DB {
	instance := &DB{dbFilename: "inventory.db"}
	instance.Init()
	return instance
}

func (db *DB) Init() error {
	log.Debug("Database Init")
	var err error
	db.db, err = sql.Open("sqlite3", db.dbFilename)
	if err != nil {
		LogAndQuit("Unable to open database file", errors.New(db.dbFilename))
	}

	tx, err := db.db.Begin()
	if err != nil {
		LogAndQuit("Initializing DB", err)
	}

	stmt := `
	CREATE TABLE IF NOT EXISTS
		AWSInstance (
			Account TEXT,
			AMI TEXT,
			ENV TEXT,
			ID TEXT,
			KeypairName TEXT,
			LaunchTime DATETIME,
			Name TEXT,
			Notes TEXT,
			OS TEXT,
			PrivateIP TEXT,
			PublicIP TEXT,
			Region TEXT,
			Size TEXT,
			Skip INTEGER,
			SSHKey TEXT,
			SSHPort TEXT,
			State TEXT,
			Subnet TEXT,
			User TEXT,
			VPC TEXT,
			LastSeen DATETIME
		)`
	_, err = tx.Exec(stmt)
	if err != nil {
		LogAndQuit("Unable to create table (Instance)", err)
	}
	
	stmt = `
	CREATE TABLE IF NOT EXISTS
		Tags (
			InstanceID TEXT,
			Key TEXT,
			Value TEXT
		)`
	_, err = tx.Exec(stmt)
	if err != nil {
		LogAndQuit("Unable to create table (Tags)", err)
	}

	err = tx.Commit()
	if err != nil {
		LogAndQuit("Error committing initialization changes", err)
	}

	return nil
}

func (db *DB) InstanceExists(i Instance) (bool) {
	var count int

	tx, err := db.db.Begin()
	if err != nil {
		LogAndQuit("Error checking for instance in DB", err)
	}

	stmt := `
	SELECT
		count(*)
	FROM
		AWSInstance
	WHERE
		ID = ? AND
		State != 'terminated'`
	err = tx.QueryRow(stmt, i.ID, i.State).Scan(&count)
	if err != nil {
		LogAndQuit("Error pulling instance from DB", err)
	}
	
	return count > 0
}

func (db *DB) UpdateInstance(i Instance) {
	tx, err := db.db.Begin()
	if err != nil {
		LogAndQuit(fmt.Sprintf("Unable to update instance: %s", i.ID), err)
	}
	defer tx.Commit()

	stmt := `
	UPDATE AWSInstance SET
		ENV = ?,
		Name = ?,
		OS = ?,
		PrivateIP = ?,
		PublicIP = ?,
		Size = ?,
		Skip = ?,
		SSHKey = ?,
		SSHPort = ?,
		State = ?,
		Subnet = ?,
		User = ?
	WHERE
		ID = ?`
	tx.Exec(stmt, i.ENV, i.Name, i.OS, i.PrivateIP, i.PublicIP, i.Size, i.Skip, i.SSHKey, i.SSHPort, i.State, i.Subnet, i.User, i.ID)
}

func (db *DB) FlagInstancesAsTerminated(needsMarked []string) error {
	tx, err := db.db.Begin()
	if err != nil {
		LogAndQuit("Unable to flag instances as terminated", err)
	}
	defer tx.Commit()
	for _, id := range needsMarked {
		stmt := "UPDATE Instance SET State = 'terminated' WHERE ID = ?"
		tx.Exec(stmt, id)
	}
	return nil
}

func (db *DB) GetActiveInstances() ([]string, error) {
	log.Debug("Getting active instances.")
	tx, err := db.db.Begin()
	if err != nil {
		return make([]string, 0), err
	}
	defer tx.Commit()

	instances := make([]string, 0)
	
	stmt := `SELECT InstanceID FROM AWSInstance WHERE State != 'terminated'`
	rows, err := tx.Query(stmt)
	if err != nil {
		return make([]string, 0), err
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		err := rows.Scan(&id)
		if err != nil {
			panic(err)
		}
		instances = append(instances, id)
	}
	log.Debug("Got all active instances.")
	return instances, nil
}

func (db *DB) AddInstancesToDB(instances map[string]Instance) {
	log.Debug("Adding instances to DB.")
	for instanceID, instance := range instances {
		log.Debugf("Adding %s to db\n", instanceID)
		db.AddOrUpdateInstance(instance)
	}
}

func (db *DB) AddOrUpdateInstance(i Instance) {
	if db.InstanceExists(i) {
		log.Debug("Updating.")
		db.UpdateInstance(i)
		log.Debug("Updated.")
	} else {
		log.Debug("Adding.")
		db.AddInstance(i)
		log.Debug("Added")
	}
}

func (db *DB) AddInstance(i Instance) {
	log.Debug("Beginning of AddInstance.")
	tx, err := db.db.Begin()
	if err != nil {
		LogAndQuit("Unable to add instance", err)
	}
	//defer tx.Commit()

	log.Debug("Preparing.")
	stmt := `
	INSERT INTO AWSInstance (
		Account, AMI, ENV, ID, KeypairName, LaunchTime, Name, Notes, OS, PrivateIP, PublicIP,
		Region, Size, SSHKey, SSHPort, State, Subnet, User, VPC, LastSeen
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	log.Debug("Executing.")
	_, err = tx.Exec(stmt,
		i.Account, i.AMI, i.ENV, i.ID, i.KeypairName, i.LaunchTime, i.Name, i.Notes, i.OS, i.PrivateIP, i.PublicIP,
		i.Region, i.Size, i.SSHKey, i.SSHPort, i.State, i.Subnet, i.User, i.VPC, time.Now(),
	)
	if err != nil {
		LogAndQuit("Unable to insert instance", err)
	}

	log.Debug("Committing.")
	err = tx.Commit()
	if err != nil {
		log.Debugf("Error committing: %v\n", err)
	}
	log.Debug("Committed.")
}

func (db *DB) DeleteTags(ID string) {
	tx, err := db.db.Begin()
	if err != nil {
		LogAndQuit("Unable to begin transaction (DeleteTags)", err)
	}
	defer tx.Commit()

	stmt := "DELETE FROM Tags WHERE InstanceID = ?"
	_, err = tx.Exec(stmt, ID)
	if err != nil {
		LogAndQuit("Unable to delete tag", err)
	}
}

func (db *DB) AddTags(i Instance) {
	// Init DB
	tx, err := db.db.Begin()
	if err != nil {
		LogAndQuit("Unable to begin transaction (AddTags)", err)
	}
	defer tx.Commit()

	// Delete old tags
	db.DeleteTags(i.ID)
	
	// Add new tags
	for k, v := range i.Tags {
		stmt := "INSERT INTO Tags (InstanceID, Key, Value) VALUES (?, ?, ?)"
		_, err := tx.Exec(stmt, i.ID, k, v)
		if err != nil {
			LogAndQuit("Unable to insert tag", err)
		}
	}
}

func Pause() {
	var userInput string
	fmt.Printf("Waiting for user input... <Enter>")
	fmt.Scanln(&userInput)
	fmt.Printf("\n")
}
