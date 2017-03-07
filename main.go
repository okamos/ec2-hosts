package main

import (
	"bufio"
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const ConfigPath = "config/ec2-hosts.tml"
const HostsPath = "/etc/hosts"

const Interval = 30 * time.Second

var config Config
var loopFlag bool

type Config struct {
	Aws awsParams
	// EC2 instance tag table
	Tags map[string]interface{}
}

type awsParams struct {
	Region          string `toml:"region"`
	AccessKeyId     string `toml:"access_key_id"`
	SecretAccessKey string `toml:"secret_access_key"`
}

func parseOptions() {
	flag.BoolVar(&loopFlag, "loop", false, "Exec unlimited loop. If you want to exec as a real daemon process, use service components like systemd, supervisord and bg.")
	flag.Parse()
}

func parseConfig() map[string][]string {
	asset, err := Asset(ConfigPath)
	if err != nil {
		panic(err)
	}

	err = toml.Unmarshal(asset, &config)
	if err != nil {
		panic(err)
	}

	ret := map[string][]string{}

	for tag, rawValue := range config.Tags {
		var values []string
		parseValue(rawValue, &values)
		ret[tag] = values
	}

	return ret
}

func parseValue(v interface{}, ret *[]string) {
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

func updateHosts(hostsTable map[string]string) {
	hosts, err := os.Open(HostsPath)
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
		if len(chunk) > 0 {
			hostNames := chunk[1:]
			updated := false

			for _, hostName := range hostNames {
				if newIpAddress, ok := hostsTable[hostName]; ok {
					newHostsWriter.WriteString(newIpAddress + " " + strings.Join(hostNames, " ") + "\n")
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

	err = os.Rename(newHosts.Name(), HostsPath)
	if err != nil {
		panic(err)
	}
	err = os.Chmod(HostsPath, 0644)
	if err != nil {
		panic(err)
	}
}

func describeInstances(tag string, values []string) map[string]string {
	s, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	ec2Client := ec2.New(s, &aws.Config{
		Region:      aws.String(config.Aws.Region),
		Credentials: credentials.NewStaticCredentials(config.Aws.AccessKeyId, config.Aws.SecretAccessKey, ""),
	})

	ret := map[string]string{}

	for _, value := range values {
		params := &ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("tag:" + tag),
					Values: []*string{
						aws.String(value),
					},
				},
			},
		}

		resp, err := ec2Client.DescribeInstances(params)
		if err != nil {
			panic(err)
		}

		// use first instance
		if len(resp.Reservations) > 0 {
			reservation := resp.Reservations[0]
			if len(reservation.Instances) > 0 {
				instance := reservation.Instances[0]
				// use tag value as hostname
				ret[value] = *instance.PrivateIpAddress
			}
		}
	}

	return ret
}

func exec(tagsTable map[string][]string) {
	hostsTable := map[string]string{} // hostname : ipAddress

	for tag, values := range tagsTable {
		for hostName, ipAddress := range describeInstances(tag, values) {
			hostsTable[hostName] = ipAddress
		}
	}

	updateHosts(hostsTable)
}

func main() {
	parseOptions()

	tagsTable := parseConfig() // tag : [value, ...]

	if loopFlag {
		exec(tagsTable)
		ticker := time.Tick(Interval)
		for range ticker {
			exec(tagsTable)
		}
	} else {
		exec(tagsTable)
	}
}
