package inventoryengine

import (
	"context"
	"sync"
	//"github.com/aws/aws-sdk-go-v2/aws"
	"errors"
	"strings"

	invconfig "github.com/ascheel/goinventory/inventory/config"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

type AWS struct {
	Profile string
	Region string
	EC2 ec2.Client
	Instances map[string]Instance
	db *DB
}

var awsInstance *AWS
var awsOnce sync.Once

func NewAWS(db *DB) *AWS {
	// Our Singleton
	awsOnce.Do(func() {
		awsInstance = &AWS{}
		awsInstance.db = db
	})
	return awsInstance
}

type EC2DescribeInstancesAPI interface {
	DescribeInstances(
		ctx context.Context,
		params *ec2.DescribeInstancesInput,
		optFns ...func(*ec2.Options),
	) (*ec2.DescribeInstancesOutput, error)
}

func GetInstances(c context.Context, api EC2DescribeInstancesAPI, input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error){
	return api.DescribeInstances(c, input)
}

func (a *AWS) RefreshInstances() {
	a.Instances = make(map[string]Instance)
	a.GetInstances()
}

func (a *AWS) GetInstanceList() ([]string) {
	instances := make([]string, 0)
	for id := range a.Instances {
		instances = append(instances, id)
	}
	return instances
}

func (a *AWS) AddInstancesToDB() {
	log.Debug("Adding instances to DB.")
	for instanceID, instance := range a.Instances {
		log.Debugf("Adding %s to db\n", instanceID)
		a.db.AddOrUpdateInstance(instance)
	}
}

func (a *AWS) GetInstances() {
	log.Debug("Getting instances.")
	c := invconfig.NewConfig("de_test.yml")

	if a.Instances == nil {
		a.Instances = make(map[string]Instance)
		var profile string
		var region string

		count := 0
		for profile = range c.AWS.Accounts {
			log.Infof("Checking profile %s\n", profile)
			for region = range c.AWS.Regions {
				log.Infof("Checking region %s\n", region)
				cfg, err := config.LoadDefaultConfig(
					context.TODO(),
					config.WithRegion(region),
					config.WithSharedConfigProfile(profile),
				)
				if err != nil {
					return
				}
				client := ec2.NewFromConfig(cfg)
				input := &ec2.DescribeInstancesInput{}

				result, err := GetInstances(context.TODO(), client, input)
				if err != nil {
					log.Fatalf("Unable to get instances: %v\n", err)
				}
				for _, r := range result.Reservations {
					for _, i := range r.Instances {
						count += 1
						log.Debugf("Found instance: %s (%d)\n", *i.InstanceId, count)
						_instance := a.TranslateInstance(i)
						_instance.Account = profile
						_instance.Region = region
						_instance.ENV = c.AWS.Accounts[profile].Env
						a.Instances[_instance.ID] = _instance
					}
				}
			}
		}
	}
}

func (a *AWS) TranslateInstance(instance types.Instance) Instance {
	name, err := GetTag(instance.Tags, "Name")
	if err != nil {
		name = ""
	}
	keyname := ""
	if instance.KeyName != nil {
		keyname = *instance.KeyName
	}
	subnet := ""
	if instance.SubnetId != nil {
		subnet = *instance.SubnetId
	}
	vpcid := ""
	if instance.VpcId != nil {
		vpcid = *instance.VpcId
	}
	i := Instance{
		ID: *instance.InstanceId,
		AMI: *instance.ImageId,
		KeypairName: keyname,
		LaunchTime: *instance.LaunchTime,
		Name: name,
		State: string(instance.State.Name),
		Subnet: subnet,
		VPC: vpcid,
	}
	return i
}

func (a *AWS) Init() error {
	a.Profile = "na-ea-dev"
	a.Region  = "us-east-1"
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return err
	}

	a.EC2 = *ec2.NewFromConfig(cfg)

	return nil
}

func GetTag(tags []types.Tag, tag string) (string, error) {
	//func GetTag(tags []map[string]string, tag string) (string) {
		for _, item := range tags {
			if strings.EqualFold(tag, *item.Key) {
				return *item.Value, nil
			}
		}
		return "", errors.New("tag does not exist")
	}
