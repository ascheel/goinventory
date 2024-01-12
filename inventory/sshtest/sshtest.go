package sshtest

import (
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"

	//"golang.org/x/crypto/ssh/knownhosts"
	"bufio"
	"fmt"
	"io"
	//"path/filepath"

	//"strconv"
	"errors"
	"net"
	"syscall"
	"time"

	//"strings"
	"crypto/sha256"
	"encoding/json"
	"reflect"

	//"flag"
)

type ConnectionInfo struct {
	Host     string
	User     string
	Port     string
	Key      string
	Password string
	ErrCode  int
	ErrText  string
	ErrRaw   error
}

var connInfo ConnectionInfo

// func init() {
// 	flag.StringVar(&connInfo.User, "user", "", "User name")
// 	flag.StringVar(&connInfo.Port, "port", "22", "Port number (default 22)")
// 	flag.StringVar(&connInfo.Host, "host", "", "Host to connect to")
// 	flag.StringVar(&connInfo.Password, "password", "", "Password")
// 	flag.StringVar(&connInfo.Key, "private-key", "", "Private key filename")
	
// 	flag.Parse()

// 	if connInfo.Password != "" && connInfo.Key != "" {
// 		log.Fatalln("Only supply password or private key, not both.")
// 	}

// 	var goodToGo bool = true

// 	if connInfo.User == "" {
// 		fmt.Println("Error: user is a required argument")
// 		goodToGo = false
// 	} else if connInfo.Host == "" {
// 		fmt.Println("Error: host is a required argument")
// 		goodToGo = false
// 	}
// 	if goodToGo == false {
// 		os.Exit(1)
// 	}
// }

func check(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func readSize(file *os.File) int64 {
	stat, err := file.Stat()
	check(err)
	return stat.Size()
}

func readFile(filename string) []byte {
	file, err := os.Open(filename)
	check(err)
	defer file.Close()
	// Read the file into a byte slice
	bs := make([]byte, readSize(file))
	_, err = bufio.NewReader(file).Read(bs)
	if err != nil && err != io.EOF {
		fmt.Println(err)
	}
	return bs
}

type ConnectionInfoer interface {
	SSHConnect()
	TryConnect()
	PrintStruct()
	GetJson()
}

func (connInfo *ConnectionInfo) SSHConnect() (*ssh.Client, *ssh.Session, error) {
	var auth     []ssh.AuthMethod

	hostString := net.JoinHostPort(connInfo.Host, connInfo.Port)

	timeout, _ := time.ParseDuration("5s")
	if len(connInfo.Password) == 0 && len(connInfo.Key) == 0 {
		log.Fatalln("Both key and password are empty.  One must be provided.")
	} else if len(connInfo.Key) > 0 {
		keyData := readFile(connInfo.Key)
		pKey, keyErr := ssh.ParsePrivateKey(keyData)
		if keyErr != nil {
			log.Fatalln("Bad key: " + connInfo.Key)
		}
		auth = []ssh.AuthMethod{ssh.PublicKeys(pKey)}
	} else if len(connInfo.Password) > 0 {
		auth = []ssh.AuthMethod{ssh.Password(connInfo.Password)}
	}

	conf := &ssh.ClientConfig{
		User: connInfo.User,
		Auth: auth,
		Timeout: timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial(
		"tcp",
		hostString,
		conf,
	)
	if err != nil {
		return nil, nil, err
	}

	session, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}

	return client, session, nil
}

func (connInfo *ConnectionInfo) TryConnect() bool {
	if len(connInfo.Key) == 0 && len(connInfo.Password) == 0 {
		log.Fatalln("No key or password provided.")
	} else if len(connInfo.Key) > 0 && len(connInfo.Password) > 0 {
		log.Fatalln("Application does not yet support passwords AND keys both.")
	} else if len(connInfo.Host) == 0 {
		log.Fatalln("No host provided.")
	}
	client, session, err := connInfo.SSHConnect()
	connInfo.ErrRaw = err
	if err != nil {
		return false
	}

	command := "ls -l /"
	_, err = session.CombinedOutput(command)
	if err != nil {
		return false
	}
	if client == nil || session == nil {
		log.Fatal("You shouldn't get here.")
		return false
	}
	return true
}

func (connInfo *ConnectionInfo) PrintStruct() {
	s := reflect.ValueOf(&connInfo).Elem().Elem()
	typeOfSSH := s.Type()
	//fmt.Println(typeOfSSH)
	for i := 0; i < s.NumField(); i++ {
	 	f := s.Field(i)
		//fmt.Println(f)
		fmt.Printf("%d: %s %s = %v\n", i, typeOfSSH.Field(i).Name, f.Type(), f.Interface())
	}
}

func (connInfo *ConnectionInfo) GetJson() string {
	type e struct {
		Host string
		Port string
		User string
		Password string
		Key string
		ErrCode int
		ErrText string
	}
	obj := e{
		Host: connInfo.Host,
		Port: connInfo.Port,
		User: connInfo.User,
		Password: "Hash: " + sha256sum(connInfo.Password),
		Key: connInfo.Key,
		ErrCode: connInfo.ErrCode,
		ErrText: connInfo.ErrText,
	}
	if len(connInfo.Password) == 0 { obj.Password = "" }
	jsonData, err := json.MarshalIndent(obj, "", "    ")
	if err != nil {
		log.Fatalln("Failed to parse json output.")
	}
	return string(jsonData)
}

func handleExit(connInfo *ConnectionInfo) {
	// 0 = success
	// 1 = authentication failure
	// 2 = timeout
	// 3 = connection refused
	// 4 = no route to host
	// 5 = can't resolve host

	//printStruct(connInfo)
	//fmt.Printf("%+v\n", connInfo)
	var dnsError *net.DNSError
	if connInfo.ErrRaw == nil {
		connInfo.ErrCode = 0
		connInfo.ErrText = "success"
	} else {
		if isAuthenticationError(connInfo.ErrRaw) {
			connInfo.ErrCode = 1
			connInfo.ErrText = "authentication failure"
		} else if os.IsTimeout(connInfo.ErrRaw) {
			connInfo.ErrCode = 2
			connInfo.ErrText = "timeout"
		} else if errors.Is(connInfo.ErrRaw, syscall.ECONNREFUSED) {
			connInfo.ErrCode = 3
			connInfo.ErrText = "connection refused"
		} else if errors.Is(connInfo.ErrRaw, dnsError) {
			connInfo.ErrCode = 5
			connInfo.ErrText = "DNS Lookup error 2"
		} else if isDNSError(connInfo.ErrRaw) {
			connInfo.ErrCode = 6
			connInfo.ErrText = "DNS Lookup error"
		} else {
			connInfo.ErrCode = 255
			connInfo.ErrText = fmt.Sprint(connInfo.ErrRaw)
		}
	}
	fmt.Println(connInfo.GetJson())
	os.Exit(connInfo.ErrCode)
}

func isDNSError(err error) bool {
	dnsKeywords := []string{
		"no such host",
	}
	for _, keyword := range dnsKeywords {
		if strings.Contains(strings.ToLower(string(fmt.Sprint(err))), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func isAuthenticationError(err error) bool {
	authKeywords := []string{
		"authentication failed",
		"invalid credentials",
		"unable to authenticate",
		"no supported methods remain",
	}
	for _, keyword := range authKeywords {
		if strings.Contains(strings.ToLower(string(fmt.Sprint(err))), strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func sha256sum(text string) string {
	hasher := sha256.New()
	hasher.Write([]byte(text))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

// func main() {
// 	tryConnect(&connInfo)
// 	handleExit(&connInfo)
// }

