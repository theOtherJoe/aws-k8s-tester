// Package ec2config defines EC2 configuration.
package ec2config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-k8s-tester/ec2config/plugins"
	ec2types "github.com/aws/aws-k8s-tester/pkg/awsapi/ec2"
	"sigs.k8s.io/yaml"
)

// Config defines EC2 configuration.
type Config struct {
	// EnvPrefix is used to update configuration via environmental variables.
	// The default is "AWS_K8S_TESTER_EC2_".
	EnvPrefix string

	// AWSAccountID is the AWS account ID.
	AWSAccountID string `json:"aws-account-id"`
	// AWSRegion is the AWS region.
	AWSRegion string `json:"aws-region"`

	// LogDebug is true to enable debug level logging.
	LogDebug bool `json:"log-debug"`

	// LogOutputs is a list of log outputs. Valid values are 'default', 'stderr', 'stdout', or file names.
	// Logs are appended to the existing file, if any.
	// Multiple values are accepted. If empty, it sets to 'default', which outputs to stderr.
	// See https://godoc.org/go.uber.org/zap#Open and https://godoc.org/go.uber.org/zap#Config for more details.
	LogOutputs []string `json:"log-outputs"`
	// LogOutputToUploadPath is the aws-k8s-tester log file path to upload to cloud storage.
	// Must be left empty.
	// This will be overwritten by cluster name.
	LogOutputToUploadPath       string `json:"log-output-to-upload-path"`
	LogOutputToUploadPathBucket string `json:"log-output-to-upload-path-bucket"`
	LogOutputToUploadPathURL    string `json:"log-output-to-upload-path-url"`
	// UploadTesterLogs is true to auto-upload log files.
	UploadTesterLogs bool `json:"upload-tester-logs"`

	// UploadBucketExpireDays is the number of days for objects in S3 bucket to expire.
	// Set 0 to not expire.
	UploadBucketExpireDays int `json:"upload-bucket-expire-days"`

	// Tag is the tag used for all cloudformation stacks.
	Tag string `json:"tag"`
	// ClusterName is an unique ID for cluster.
	ClusterName string `json:"cluster-name"`

	// WaitBeforeDown is the duration to sleep before EC2 tear down.
	// This is for "test".
	WaitBeforeDown time.Duration `json:"wait-before-down"`
	// Down is true to automatically tear down EC2 in "test".
	// Note that this is meant to be used as a flag in "test".
	// Deployer implementation should not call "Down" inside "Up" method.
	Down bool `json:"down"`

	// ConfigPath is the configuration file path.
	// If empty, it is autopopulated.
	// Deployer is expected to update this file with latest status,
	// and to make a backup of original configuration
	// with the filename suffix ".backup.yaml" in the same directory.
	ConfigPath       string    `json:"config-path"`
	ConfigPathBucket string    `json:"config-path-bucket"` // read-only to user
	ConfigPathURL    string    `json:"config-path-url"`    // read-only to user
	UpdatedAt        time.Time `json:"updated-at"`         // read-only to user

	// ImageID is the Amazon Machine Image (AMI).
	ImageID string `json:"image-id"`
	// UserName is the user name used for running init scripts or SSH access.
	UserName string `json:"user-name"`
	// Plugins is the list of plugins.
	Plugins []string `json:"plugins"`

	// InitScript contains init scripts (run-instance UserData field).
	// Script must be started with "#!/usr/bin/env bash" IF "Plugins" field is not defined.
	// And will be base64-encoded. Do not base64-encode. Just configure as plain-text.
	// Let this "ec2" package base64-encode.
	// Outputs are saved in "/var/log/cloud-init-output.log" in EC2 instance.
	// "tail -f /var/log/cloud-init-output.log" to check the progress.
	// Reference: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/user-data.html.
	// Note that if both "Plugins" and "InitScript" are not empty,
	// "InitScript" field is always appended to the scripts generated by "Plugins" field.
	InitScript string `json:"init-script"`
	// InitScriptCreated is true once the init script has been created.
	// This is to prevent redundant init script updates from plugins.
	InitScriptCreated bool `json:"init-script-created"`

	// InstanceType is the instance type.
	InstanceType string `json:"instance-type"`
	// ClusterSize is the number of EC2 instances to create.
	ClusterSize int `json:"cluster-size"`

	// KeyName is the name of the key pair used for SSH access.
	// Leave empty to create a temporary one.
	KeyName string `json:"key-name"`
	// KeyPath is the file path to the private key.
	KeyPath       string `json:"key-path"`
	KeyPathBucket string `json:"key-path-bucket"`
	KeyPathURL    string `json:"key-path-url"`
	// KeyCreateSkip is true to indicate that EC2 key pair has been created, so needs no creation.
	KeyCreateSkip bool `json:"key-create-skip"`
	// KeyCreated is true to indicate that EC2 key pair has been created, so needs be cleaned later.
	KeyCreated bool `json:"key-created"`

	// VPCCIDR is the VPC CIDR.
	VPCCIDR string `json:"vpc-cidr"`
	// VPCID is the VPC ID to use.
	// Leave empty to create a temporary one.
	VPCID string `json:"vpc-id"`
	// VPCCreated is true to indicate that EC2 VPC has been created, so needs be cleaned later.
	// Set this to false, if the VPC is reused from somewhere else, so the original VPC creator deletes the VPC.
	VPCCreated bool `json:"vpc-created"`
	// InternetGatewayID is the internet gateway ID.
	InternetGatewayID string `json:"internet-gateway-id"`
	// RouteTableIDs is the list of route table IDs.
	RouteTableIDs []string `json:"route-table-ids"`

	// SubnetIDs is a list of subnet IDs to use.
	// If empty, it will fetch subnets from a given or created VPC.
	// And randomly assign them to instances.
	SubnetIDs                  []string          `json:"subnet-ids"`
	SubnetIDToAvailabilityZone map[string]string `json:"subnet-id-to-availability-zone"` // read-only to user

	// IngressRulesTCP is a map from TCP port range to CIDR to allow via security groups.
	IngressRulesTCP map[string]string `json:"ingress-rules-tcp"`

	// SecurityGroupIDs is the list of security group IDs.
	// Leave empty to create a temporary one.
	SecurityGroupIDs []string `json:"security-group-ids"`

	// AssociatePublicIPAddress is true to associate a public IP address.
	AssociatePublicIPAddress bool `json:"associate-public-ip-address"`

	// Instances is a set of EC2 instances created from this configuration.
	Instances map[string]Instance `json:"instances"`

	// Wait is true to wait until all EC2 instances are ready.
	Wait bool `json:"wait"`

	// InstanceProfileName is the name of an instance profile with permissions to manage EC2 instances.
	InstanceProfileName string `json:"instance-profile-name"`

	// CustomScript is executed at the end of EC2 init script.
	CustomScript string `json:"custom-script"`
}

// Instance represents an EC2 instance.
type Instance struct {
	ImageID             string               `json:"image-id"`
	InstanceID          string               `json:"instance-id"`
	InstanceType        string               `json:"instance-type"`
	KeyName             string               `json:"key-name"`
	Placement           Placement            `json:"placement"`
	PrivateDNSName      string               `json:"private-dns-name"`
	PrivateIP           string               `json:"private-ip"`
	PublicDNSName       string               `json:"public-dns-name"`
	PublicIP            string               `json:"public-ip"`
	State               State                `json:"state"`
	SubnetID            string               `json:"subnet-id"`
	VPCID               string               `json:"vpc-id"`
	BlockDeviceMappings []BlockDeviceMapping `json:"block-device-mappings"`
	EBSOptimized        bool                 `json:"ebs-optimized"`
	RootDeviceName      string               `json:"root-device-name"`
	RootDeviceType      string               `json:"root-device-type"`
	SecurityGroups      []SecurityGroup      `json:"security-groups"`
	LaunchTime          time.Time            `json:"launch-time"`
}

// Placement defines EC2 placement.
type Placement struct {
	AvailabilityZone string `json:"availability-zone"`
	Tenancy          string `json:"tenancy"`
}

// State defines an EC2 state.
type State struct {
	Code int64  `json:"code"`
	Name string `json:"name"`
}

// BlockDeviceMapping defines a block device mapping.
type BlockDeviceMapping struct {
	DeviceName string `json:"device-name"`
	EBS        EBS    `json:"ebs"`
}

// EBS defines an EBS volume.
type EBS struct {
	DeleteOnTermination bool   `json:"delete-on-termination"`
	Status              string `json:"status"`
	VolumeID            string `json:"volume-id"`
}

// SecurityGroup defines a security group.
type SecurityGroup struct {
	GroupName string `json:"group-name"`
	GroupID   string `json:"group-id"`
}

// genTag generates a tag for cluster name, CloudFormation, and S3 bucket.
func genTag() string {
	// use UTC time for everything
	now := time.Now().UTC()
	return fmt.Sprintf("a8-ec2-%d%02d%02d", now.Year()-2000, int(now.Month()), now.Day())
}

// NewDefault returns a copy of the default configuration.
func NewDefault() *Config {
	vv := defaultConfig
	return &vv
}

const envPfx = "AWS_K8S_TESTER_EC2_"

// defaultConfig is the default configuration.
//  - empty string creates a non-nil object for pointer-type field
//  - omitting an entire field returns nil value
//  - make sure to check both
var defaultConfig = Config{
	EnvPrefix: envPfx,
	AWSRegion: "us-west-2",

	WaitBeforeDown: time.Minute,
	Down:           true,

	LogDebug: false,

	// default, stderr, stdout, or file name
	// log file named with cluster name will be added automatically
	LogOutputs:             []string{"stderr"},
	UploadTesterLogs:       false,
	UploadBucketExpireDays: 2,

	// TODO: use Amazon EKS-optimized AMI, https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
	// Amazon Linux 2 AMI (HVM), SSD Volume Type
	ImageID:  "ami-032509850cf9ee54e",
	UserName: "ec2-user",
	Plugins: []string{
		"update-amazon-linux-2",
		"install-start-docker-amazon-linux-2",
		// "install-kubeadm-amazon-linux-2-1.13.0",
	},

	/*
	   // Ubuntu Server 16.04 LTS, SSD Volume Type
	   ImageID:  "ami-076e276d85f524150",
	   UserName: "ubuntu",
	   Plugins: []string{
	   	"update-ubuntu",
	   	"install-start-docker-ubuntu",
	   	// "install-kubeadm-ubuntu-1.13.0",
	   },
	*/

	// 2 vCPU, 8 GB RAM
	InstanceType: "m5.large",
	ClusterSize:  1,

	AssociatePublicIPAddress: true,

	KeyCreateSkip: false,
	KeyCreated:    false,

	VPCCIDR: "192.168.0.0/16",
	IngressRulesTCP: map[string]string{
		"22": "0.0.0.0/0",
	},

	Wait: true,
}

// UpdateFromEnvs updates fields from environmental variables.
func (cfg *Config) UpdateFromEnvs() error {
	cc := *cfg

	tp, vv := reflect.TypeOf(&cc).Elem(), reflect.ValueOf(&cc).Elem()
	for i := 0; i < tp.NumField(); i++ {
		jv := tp.Field(i).Tag.Get("json")
		if jv == "" {
			continue
		}
		jv = strings.Replace(jv, ",omitempty", "", -1)
		jv = strings.Replace(jv, "-", "_", -1)
		jv = strings.ToUpper(strings.Replace(jv, "-", "_", -1))
		env := cfg.EnvPrefix + jv
		if os.Getenv(env) == "" {
			continue
		}
		sv := os.Getenv(env)

		fieldName := tp.Field(i).Name

		switch vv.Field(i).Type().Kind() {
		case reflect.String:
			vv.Field(i).SetString(sv)

		case reflect.Bool:
			bb, err := strconv.ParseBool(sv)
			if err != nil {
				return fmt.Errorf("failed to parse %q (%q, %v)", sv, env, err)
			}
			vv.Field(i).SetBool(bb)

		case reflect.Int, reflect.Int32, reflect.Int64:
			if fieldName == "WaitBeforeDown" {
				dv, err := time.ParseDuration(sv)
				if err != nil {
					return fmt.Errorf("failed to parse %q (%q, %v)", sv, env, err)
				}
				vv.Field(i).SetInt(int64(dv))
				continue
			}
			iv, err := strconv.ParseInt(sv, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse %q (%q, %v)", sv, env, err)
			}
			vv.Field(i).SetInt(iv)

		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			iv, err := strconv.ParseUint(sv, 10, 64)
			if err != nil {
				return fmt.Errorf("failed to parse %q (%q, %v)", sv, env, err)
			}
			vv.Field(i).SetUint(iv)

		case reflect.Float32, reflect.Float64:
			fv, err := strconv.ParseFloat(sv, 64)
			if err != nil {
				return fmt.Errorf("failed to parse %q (%q, %v)", sv, env, err)
			}
			vv.Field(i).SetFloat(fv)

		case reflect.Map:
			ss := strings.Split(sv, ",")
			switch fieldName {
			case "IngressRulesTCP":
				m := reflect.MakeMap(reflect.TypeOf(map[string]string{}))
				for i := range ss {
					fields := strings.Split(ss[i], "=")
					m.SetMapIndex(reflect.ValueOf(fields[0]), reflect.ValueOf(fields[1]))
				}
				vv.Field(i).Set(m)

			default:
				return fmt.Errorf("parsing field name %q not supported", fieldName)
			}

		case reflect.Slice:
			ss := strings.Split(sv, ",")
			switch fieldName {
			case "Plugins",
				"SubnetIDs",
				"SecurityGroupIDs":
				slice := reflect.MakeSlice(reflect.TypeOf([]string{}), len(ss), len(ss))
				for i := range ss {
					slice.Index(i).SetString(ss[i])
				}
				vv.Field(i).Set(slice)

			default:
				return fmt.Errorf("parsing field name %q not supported", fieldName)
			}

		default:
			return fmt.Errorf("%q (%v) is not supported as an env", env, vv.Field(i).Type())
		}
	}
	*cfg = cc

	return nil
}

// ValidateAndSetDefaults returns an error for invalid configurations.
// And updates empty fields with default values.
// At the end, it writes populated YAML to aws-k8s-tester config path.
func (cfg *Config) ValidateAndSetDefaults() (err error) {
	if len(cfg.LogOutputs) == 0 {
		return errors.New("EKS LogOutputs is not specified")
	}
	if cfg.AWSRegion == "" {
		return errors.New("empty AWSRegion")
	}
	if cfg.UserName == "" {
		return errors.New("empty UserName")
	}
	if cfg.ImageID == "" {
		return errors.New("empty ImageID")
	}

	if len(cfg.Plugins) > 0 && !cfg.InitScriptCreated {
		txt := cfg.InitScript
		cfg.InitScript, err = plugins.Create(cfg.UserName, cfg.CustomScript, cfg.Plugins)
		if err != nil {
			return err
		}
		cfg.InitScript += "\n" + txt
		cfg.InitScriptCreated = true
	}

	if cfg.InstanceType == "" {
		return errors.New("empty InstanceType")
	}
	if cfg.ClusterSize < 1 {
		return errors.New("unexpected ClusterSize")
	}

	if cfg.ClusterName == "" {
		cfg.Tag = genTag()
		cfg.ClusterName = cfg.Tag + "-" + randString(7)
	}

	if cfg.ConfigPath == "" {
		var f *os.File
		f, err = ioutil.TempFile(os.TempDir(), "a8-ec2config")
		if err != nil {
			return err
		}
		cfg.ConfigPath, _ = filepath.Abs(f.Name())
		f.Close()
		os.RemoveAll(cfg.ConfigPath)
	}
	cfg.ConfigPathBucket = filepath.Join(cfg.ClusterName, "a8-ec2config.yaml")

	cfg.LogOutputToUploadPath = filepath.Join(os.TempDir(), fmt.Sprintf("%s.log", cfg.ClusterName))
	logOutputExist := false
	for _, lv := range cfg.LogOutputs {
		if cfg.LogOutputToUploadPath == lv {
			logOutputExist = true
			break
		}
	}
	if !logOutputExist {
		// auto-insert generated log output paths to zap logger output list
		cfg.LogOutputs = append(cfg.LogOutputs, cfg.LogOutputToUploadPath)
	}
	cfg.LogOutputToUploadPathBucket = filepath.Join(cfg.ClusterName, "a8-ec2.log")

	if cfg.KeyName == "" {
		cfg.KeyName = cfg.ClusterName
	}
	cfg.KeyPathBucket = filepath.Join(cfg.ClusterName, "a8-ec2.key")
	if cfg.KeyPath == "" {
		var f *os.File
		f, err = ioutil.TempFile(os.TempDir(), "a8-ec2.key")
		if err != nil {
			return err
		}
		cfg.KeyPath, _ = filepath.Abs(f.Name())
		f.Close()
		os.RemoveAll(cfg.KeyPath)
	}

	if _, ok := ec2types.InstanceTypes[cfg.InstanceType]; !ok {
		return fmt.Errorf("unexpected InstanceType %q", cfg.InstanceType)
	}

	return nil
}

// Load loads configuration from YAML.
//
// Example usage:
//
//  import "github.com/aws/aws-k8s-tester/internal/ec2/config"
//  cfg := config.Load("test.yaml")
//  p, err := cfg.BackupConfig()
//  err = cfg.ValidateAndSetDefaults()
//
// Do not set default values in this function.
// "ValidateAndSetDefaults" must be called separately,
// to prevent overwriting previous data when loaded from disks.
func Load(p string) (cfg *Config, err error) {
	var d []byte
	d, err = ioutil.ReadFile(p)
	if err != nil {
		return nil, err
	}
	cfg = new(Config)
	if err = yaml.Unmarshal(d, cfg); err != nil {
		return nil, err
	}

	if cfg.Instances == nil {
		cfg.Instances = make(map[string]Instance)
	}

	if !filepath.IsAbs(cfg.ConfigPath) {
		cfg.ConfigPath, err = filepath.Abs(p)
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// SSHCommands returns the SSH commands.
func (cfg *Config) SSHCommands() (s string) {
	s = fmt.Sprintf(`
# change SSH key permission
chmod 400 %s
`, cfg.KeyPath)

	for _, v := range cfg.Instances {
		s += fmt.Sprintf(`# ssh into the machine
ssh -o "StrictHostKeyChecking no" -i %s %s@%s
# download to local machine
scp -i %s %s@%s:REMOTE_FILE_PATH LOCAL_FILE_PATH
# upload to remote machine
scp -i %s LOCAL_FILE_PATH %s@%s:REMOTE_FILE_PATH

`,
			cfg.KeyPath, cfg.UserName, v.PublicDNSName,
			cfg.KeyPath, cfg.UserName, v.PublicDNSName,
			cfg.KeyPath, cfg.UserName, v.PublicDNSName,
		)
	}

	return s + "\n"
}

// Sync persists current configuration and states to disk.
func (cfg *Config) Sync() (err error) {
	if !filepath.IsAbs(cfg.ConfigPath) {
		cfg.ConfigPath, err = filepath.Abs(cfg.ConfigPath)
		if err != nil {
			return err
		}
	}
	cfg.UpdatedAt = time.Now().UTC()
	var d []byte
	d, err = yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cfg.ConfigPath, d, 0600)
}

// BackupConfig stores the original aws-k8s-tester configuration
// file to backup, suffixed with ".backup.yaml".
// Otherwise, deployer will overwrite its state back to YAML.
// Useful when the original configuration would be reused
// for other tests.
func (cfg *Config) BackupConfig() (p string, err error) {
	var d []byte
	d, err = ioutil.ReadFile(cfg.ConfigPath)
	if err != nil {
		return "", err
	}
	p = fmt.Sprintf("%s.%X.backup.yaml",
		cfg.ConfigPath,
		time.Now().UTC().UnixNano(),
	)
	return p, ioutil.WriteFile(p, d, 0600)
}

const ll = "0123456789abcdefghijklmnopqrstuvwxyz"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		rand.Seed(time.Now().UTC().UnixNano())
		b[i] = ll[rand.Intn(len(ll))]
	}
	return string(b)
}
