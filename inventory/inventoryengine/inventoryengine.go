package inventoryengine

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"
	"slices"
	"sync"

	"github.com/ascheel/goinventory/inventory/config"
	"github.com/ascheel/goinventory/inventory/sshtest"
	"github.com/op/go-logging"
)

var log            = logging.MustGetLogger("inventory")

type Inventory struct {
	Instances map[string]Instance `yaml:"instances" json:"instances"`
	Report struct {
		terminated []string
		new []string
	}
	Metadata struct {
		Count map[string]int `yaml:"count" json:"count"`
		Timestamp string `yaml:"timestamp" json:"timestamp"`
	}
	db *DB
}

var inv *Inventory
var invOnce sync.Once

func NewInventory() *Inventory {
	// Our Singleton
	invOnce.Do(func() {
		inv = &Inventory{}
		inv.db = NewDB()
	})
	return inv
}

func (i *Inventory) Roll() error {
	format           := logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfile} ▶ %{level:.5s} %{id:03x}%{color:reset} %{message}`)
	//format           := logging.MustStringFormatter(`%{color}%{time:15:04:05.000} %{shortfunc} ▶ %{level:.5s} %{id:03x}%{color:reset} %{message}`)
	backend          := logging.NewLogBackend(os.Stderr, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)
	logging.SetBackend(backendFormatter)
	logging.SetLevel(logging.DEBUG, "")

	// Now get current state from AWS.
	i.ReadInventoryFromAWS()

	// Now check which instances are gone.
	i.MarkTerminated()

	// Now export the results to a file.
	i.ExportToFile()

	return nil
}

func (i *Inventory) MarkTerminated() error {
	log.Debug("Marking terminated.")
	// Find instances that no longer exist and change their state to "terminated"

	// 1) Get AWS instances
	// 2) Get database instances that are not terminated
	// 3) Any from DB that do not exist in AWS instances, flag as terminated

	// Currently on AWS
	a := NewAWS()
	instances := a.GetInstanceList()

	// Currently in Database
	notTerminated, err := i.db.GetActiveInstances()
	if err != nil {
		return err
	}

	// Now compare and flag.
	needsMarked := make([]string, 0)
	for _, id1 := range notTerminated {
		if ! slices.Contains[[]string, string](instances, id1) {
			needsMarked = append(needsMarked, id1)
		}
	}
	i.db.FlagInstancesAsTerminated(needsMarked)

	return nil
}

func DirExists(dirname string) bool {
	if _, err := os.Stat(dirname); !os.IsNotExist(err) {
		return true
	} else {
		return false
	}
}

func CreateDirIfNotExists(dirname string) error {
	if DirExists(dirname) {
		return nil
	}
	err := os.MkdirAll(dirname, 0700)
	return err
}

func (i *Inventory) GetKeys() []string {
	var keys []string
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Unable to get home directory.")
		os.Exit(1)
	}
	sshdir := path.Join(homedir, "ansible", "keys")
	
	// Create SSH Dir if not exists
	err = CreateDirIfNotExists(sshdir)
	if err != nil {
		log.Fatalf("Unable to create ssh dir: %v\n", err)
	}

	files := GetFiles(sshdir)
	for _, file := range files {
		result, err := IsPrivateKeyFile(file)
		if err != nil {
			log.Fatalf("Unable to determine if a file is a private key: %v\n", err)
		}
		if result {
			keys = append(keys, file)
		}
	}
	return keys
}

func IsPrivateKeyFile(filename string) (bool, error) {
	shortname := path.Base(filename)
	pattern := `id_{dsa|rsa|ecdsa|ed25519}.*`
	matched, err := regexp.MatchString(pattern, shortname)
	if err != nil {
		return false, err
	}
	return matched, nil
}

func GetFiles(dirname string) []string {
	var files []string
	entries, err := os.ReadDir(dirname)
	if err != nil {
		log.Fatalf("Unable to read directory %s: %v\n", err)
	}

	for _, file := range entries {
		if !file.IsDir() {
			files = append(files, file.Name())
		}
	}
	return files
}

func (i *Inventory) AddNew() error {
	// Add basic data to i.Instances
	a := NewAWS()
	for key, instance := range a.Instances {
		_, ok := i.Instances[key]
		if ok {
			// Key does exist.  It's not new.
			continue
		} else {
			// Key does not yet exist.  Add it.
			i.Instances[key] = instance
			i.Report.new = append(i.Report.new, key)
		}
	}
	// Populate login details, if known.
	c := config.Settings{}
	users := c.Inventory.Users
	sshkeys := i.GetKeys()
	fmt.Printf("Found SSH keys: %s\n", strings.Join(sshkeys, ", "))

	for _, instanceId := range i.Report.new {
		instance := a.Instances[instanceId]
		// ip_public := instance.PublicIP
		// ip_private := instance.PrivateIP
		address, err := instance.GetConnectionAddress()
		port := instance.GetPort()

		if err != nil {
			panic("Failure getting address.")
		}
		for _, user := range users {
			conn := sshtest.ConnectionInfo {
				Host: address,
				User: user,
				Port: port,
				Key: instance.SSHKey,
			}
			fmt.Println("Test")
			result := conn.TryConnect()
			if result {
				// SUCCESS!
				// First we get a copy of the map
				if entry, ok := i.Instances[instanceId]; ok {
					entry.User = user
					entry.SSHPort = port
					entry.SSHKey = instance.SSHKey
				}
			}
		}
	}
	return nil
}

func (i *Inventory) ExportToFile() error {
	// Now do the exporty stuff.
	log.Debug("Exporting to file (not yet implemented).")
	return nil
}

func (i *Inventory) AddInstancesToDB(instances map[string]Instance) {
	log.Debug("Adding instances to DB.")
	for instanceID, instance := range instances {
		log.Debugf("Adding %s to db\n", instanceID)
		i.db.AddOrUpdateInstance(instance)
	}
}

func (i *Inventory) ReadInventoryFromAWS() {
	log.Debug("Reading inventory from AWS.")
	a := NewAWS()
	instances := a.GetInstances()
	i.AddInstancesToDB(instances)
	//a.AddInstancesToDB()
}

func (i *Inventory) PrettyPrintInventory() error {
	printData, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		return err
	}
	fmt.Println(string(printData))
	return nil
}

func LogAndQuit(text string, err error) {
	msg := fmt.Sprintf("%s: %v\n", text, err)
	log.Fatal(msg)
}
