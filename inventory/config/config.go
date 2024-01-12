package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Settings struct {
	SSH struct {
		LdapGroups []string `yaml:"ldap_groups" json:"ldap_groups"`
	} `yaml:"ssh" json:"ssh"`
	AWS struct {
		Accounts map[string]struct {
			AccountNo string `yaml:"accountno" json:"accountno"`
			Env string `yaml:"env" json:"env"`
			FalconEnv string `yaml:"falcon-env" json:"falcon-env"`
			JumpHosts map[string]string `yaml:"jump_hosts" json:"jump_hosts"`
			LDAP map[string]string `yaml:"ldap" json:"ldap"`
			SplunkDir string `yaml:"splunk_dir" json:"splunk_dir"`
		} `yaml:"accounts" json:"accounts"`
		Regions map[string] struct {
			DefaultValue bool `yaml:"default" json:"default"`
			Short string `yaml:"short" json:"short"`
			Location string `yaml:"location" json:"location"`
		} `yaml:"regions" json:"regions"`
	} `yaml:"aws" json:"aws"`
	Azure struct {
		Accounts map[string] struct {
			Id string `yaml:"id" json:"id"`
			Name string `yaml:"name" json:"name"`
		} `yaml:"accounts" json:"accounts"`
		Regions map[string] struct {
			DefaultValue bool `yaml:"default" json:"default"`
			Short string `yaml:"short" json:"short"`
			Location string `yaml:"location" json:"location"`
		} `yaml:"regions" json:"regions"`
	} `yaml:"azure" json:"azure"`
	CMDB map[string]string `yaml:"cmdb" json:"cmdb"`
	Inventory struct {
		Datadir string `yaml:"datadir" json:"datadir"`
		SkipOnNoCreds bool `yaml:"skip_on_no_creds" json:"skip_on_no_creds"`
		MaxBackups int `yaml:"max_backups" json:"max_backups"`
		ExplicitKeys bool `yaml:"explicit_keys" json:"explicit_keys"`
		Ec2_required_tags []string `yaml:"ec2_required_tags" json:"ec2_required_tags"`
		Users []string `yaml:"users" json:"users"`
		FirstAttempts []string `yaml:"first_attempts" json:"first_attempts"`
		KeyBlacklist []string `yaml:"key_blacklist" json:"key_blacklist"`
		Keys []string `yaml:"keys" json:"keys"`
		OsMap map[string]string `yaml:"os_map" json:"os_map"`
	} `yaml:"inventory" json:"inventory"`
	Proxies map[string] struct{
		Description string `yaml:"description" json:"description"`
		Host string `yaml:"host" json:"host"`
		User string `yaml:"user" json:"user"`
		Key string `yaml:"key" json:"key"`
		Port string `yaml:"port" json:"port"`
		KeyVault string `yaml:"key_vault" json:"key_vault"`
	} `yaml:"proxies" json:"proxies"`
}

// func printLine() {
// 	_, _, line, _ := runtime.Caller(1)
// 	fmt.Printf("Line: %d\n", line)
// }

func parseTilde(filename string) (string) {
	if strings.HasPrefix(filename, "~/") {
		dirname, err := os.UserHomeDir()
		if err != nil {
			msg := fmt.Sprintf("Unable to obtain home directory: %s\n", err)
			panic(msg)
		}
		filename = filepath.Join(dirname, filename[2:])
	}
	return filename
}

func NewConfig(configFile string) (*Settings) {
	var err error
	var fullConfigFilename string
	var yamlFileContents []byte
	var s Settings

	fullConfigFilename, err = filepath.Abs(configFile)
	if err != nil {
		//  Panic.
		msg := fmt.Sprintf("Unable to get config file path: %s\n", err)
		panic(msg)
	}
	fullConfigFilename = parseTilde(fullConfigFilename)
	
	yamlFileContents, err = os.ReadFile(fullConfigFilename)
	if err != nil {
		msg := fmt.Sprintf("Unable to read config file: %s\n", err)
		panic(msg)
	}

	err = yaml.Unmarshal(yamlFileContents, &s)
	if err != nil {
		msg := fmt.Sprintf("Unable to parse config file: %s\n", err)
		panic(msg)
	}
	return &s
}

type Settingser interface {
	GetSettings()
	GetSettingsFile()
}
