package inventoryengine

import (
	"errors"
	"time"
)

type Instance struct {
	Account       string            `yaml:"account" json:"account"`
	AMI           string            `yaml:"ami" json:"ami"`
	CloudProvider string            `yaml:"cloud_provider" json:"cloud_provider"`
	ENV           string            `yaml:"env" json:"env"`
	ID            string            `yaml:"id" json:"id"`
	KeypairName   string            `yaml:"keypair_name" json:"keypair_name"`
	LaunchTime    time.Time         `yaml:"launch_time" json:"launch_time"`
	Name          string            `yaml:"name" json:"name"`
	Notes         string            `yaml:"notes" json:"notes"`
	OS            string            `yaml:"os" json:"os"`
	PrivateIP     string            `yaml:"private_ip" json:"private_ip"`
	PublicIP      string            `yaml:"public_ip" json:"public_ip"`
	Region        string            `yaml:"region" json:"region"`
	Size          string            `yaml:"size" json:"size"`
	Skip          bool              `yaml:"skip" json:"skip"`
	SSHKey        string            `yaml:"ssh_key" json:"ssh_key"`
	SSHPort       string            `yaml:"ssh_port" json:"ssh_port"`
	State         string            `yaml:"state" json:"state"`
	Subnet        string            `yaml:"subnet" json:"subnet"`
	Tags          map[string]string `yaml:"tags" json:"tags"`
	User          string            `yaml:"user" json:"user"`
	VPC           string            `yaml:"vpc" json:"vpc"`
}

func (instance *Instance) GetConnectionAddress() (string, error) {
	private := instance.PrivateIP
	public  := instance.PublicIP
	if len(public) == 0 && len(private) == 0 {
		return "", errors.New("no Private or Public IP address.  Is the instance being terminated?")
	} else if len(public) == 0 {
		return private, nil
	} else {
		return public, nil
	}
}

func (instance *Instance) GetPort() (string) {
	if instance.SSHPort != "" {
		return instance.SSHPort
	} else {
		return "22"
	}
}

