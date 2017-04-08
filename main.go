package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	configPath = "/etc/ec2-hosts/ec2-hosts.toml"
	hostsPath  = "/etc/hosts"
	interval   = 30 * time.Second
)

var (
	conf       config
	loopFlag   bool
	configFlag string
)

type config struct {
	Aws awsParams
	// EC2 instance tag table
	Tags map[string]interface{}
	// EC2 instance group tag table
	GroupTags map[string]interface{}
}

type awsParams struct {
	Region          string `toml:"region"`
	AccessKeyID     string `toml:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key"`
}

// for sorting ec2 instnaces
type ec2Instances []*ec2.Instance

func (xs ec2Instances) Len() int {
	return len(xs)
}

func (xs ec2Instances) Less(i, j int) bool {
	return xs[i].LaunchTime.Unix() < xs[j].LaunchTime.Unix()
}

func (xs ec2Instances) Swap(i, j int) {
	xs[i], xs[j] = xs[j], xs[i]
}

func parseOptions() {
	flag.BoolVar(&loopFlag, "loop", false, "Exec unlimited loop. If you want to exec as a real daemon process, use service components like systemd, supervisord and bg.")
	flag.StringVar(&configFlag, "config", configPath, "Set ec2-hosts config path.")
	flag.Parse()
}

func parseConfig() {
	c, err := ioutil.ReadFile(configFlag)
	if err != nil {
		panic(err)
	}

	err = toml.Unmarshal(c, &conf)
	if err != nil {
		panic(err)
	}
}

func parseTagsTable(tagsTable map[string]interface{}) map[string][]string {
	ret := map[string][]string{}

	var parseValue func(v interface{}, ret *[]string)
	parseValue = func(v interface{}, ret *[]string) {
		switch v.(type) {
		case string:
			*ret = append(*ret, v.(string))
			return
		case []interface{}:
			for _, rawValue := range v.([]interface{}) {
				parseValue(rawValue, ret)
			}
		default:
			panic("not supported type")
		}
	}

	for tag, rawValue := range tagsTable {
		var values []string
		parseValue(rawValue, &values)
		ret[tag] = values
	}

	return ret
}

func updateHosts(hostsTable map[string]string) {
	hosts, err := os.Open(hostsPath)
	if err != nil {
		panic(err)
	}
	newHosts, err := ioutil.TempFile("", "")
	if err != nil {
		panic(err)
	}

	hostsScanner := bufio.NewScanner(hosts)
	newHostsWriter := bufio.NewWriter(newHosts)

	for hostsScanner.Scan() {
		line := hostsScanner.Text()
		chunk := strings.FieldsFunc(line, func(r rune) bool { return string(r) == " " || string(r) == "\t" })
		if len(chunk) > 1 {
			ipAddress := chunk[0]
			hostNames := chunk[1:]
			updated := false

			for _, hostName := range hostNames {
				if newIPAddress, ok := hostsTable[hostName]; ok {
					if ipAddress == "127.0.0.1" {
						// ignore own host
						newHostsWriter.WriteString(line + "\n")
					} else {
						newHostsWriter.WriteString(newIPAddress + " " + strings.Join(hostNames, " ") + "\n")
					}
					delete(hostsTable, hostName)
					updated = true
					break
				}
			}

			// if not exists hosts table, write as it is
			if !updated {
				newHostsWriter.WriteString(line + "\n")
			}
		} else {
			newHostsWriter.WriteString(line + "\n")
		}
	}

	for hostName, ipAddress := range hostsTable {
		newHostsWriter.WriteString(ipAddress + " " + hostName + "\n")
	}

	newHostsWriter.Flush()

	hosts.Close()
	newHosts.Close()

	err = os.Rename(newHosts.Name(), hostsPath)
	if err != nil {
		panic(err)
	}
	err = os.Chmod(hostsPath, 0644)
	if err != nil {
		panic(err)
	}
}

func describeInstances(tag string, values []string) ec2Instances {
	s, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	ec2Client := ec2.New(s, &aws.Config{
		Region:      aws.String(conf.Aws.Region),
		Credentials: credentials.NewStaticCredentials(conf.Aws.AccessKeyID, conf.Aws.SecretAccessKey, ""),
	})

	var ret ec2Instances

	for _, value := range values {
		params := &ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("tag:" + tag),
					Values: []*string{
						aws.String(value),
					},
				},
				{
					Name: aws.String("instance-state-name"),
					Values: []*string{
						aws.String("running"),
					},
				},
			},
		}

		resp, err := ec2Client.DescribeInstances(params)
		if err != nil {
			panic(err)
		}

		for _, r := range resp.Reservations {
			ret = append(ret, r.Instances...)
		}
	}

	sort.Sort(ret)

	return ret
}

func exec(tagsTable, groupTagsTable map[string][]string) {
	hostsTable := map[string]string{} // hostName : ipAddress

	getHostname := func(instance *ec2.Instance, tagKey string) string {
		for _, tag := range instance.Tags {
			if *tag.Key == tagKey {
				return *tag.Value
			}
		}
		return ""
	}
	getPrivateIPAddress := func(instance *ec2.Instance) string {
		return *instance.PrivateIpAddress
	}

	for tag, values := range tagsTable {
		instances := describeInstances(tag, values)
		for _, instance := range instances {
			if instance != nil {
				hostName := getHostname(instance, tag)
				if _, ok := hostsTable[hostName]; !ok && len(hostName) > 0 {
					hostsTable[hostName] = getPrivateIPAddress(instance)
				}
			}
		}
	}

	for tag, values := range groupTagsTable {
		instances := describeInstances(tag, values)
		for _, instance := range instances {
			if instance != nil {
				hostName := getHostname(instance, "Name")
				if _, ok := hostsTable[hostName]; !ok && len(hostName) > 0 {
					hostsTable[hostName] = getPrivateIPAddress(instance)
				}
			}
		}
	}

	updateHosts(hostsTable)
}

func main() {
	parseOptions()
	parseConfig()

	tagsTable := parseTagsTable(conf.Tags)           // tag : [value, ...]
	groupTagsTable := parseTagsTable(conf.GroupTags) // tag : [value, ...]

	if loopFlag {
		exec(tagsTable, groupTagsTable)
		ticker := time.Tick(interval)
		for range ticker {
			exec(tagsTable, groupTagsTable)
		}
	} else {
		exec(tagsTable, groupTagsTable)
	}
}
