// quake2 server resolver is a cli utility for pulling data from a text-format
// protobuf implimenting github.com/packetflinger/libq2/servers_file.proto
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	pb "github.com/packetflinger/libq2/proto"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	serversFile = ".q2servers.config" // default name, should be in home directory
	config      = flag.String("config", "", "Specify a server data file")
	//property    = flag.String("property", "", "Output only this specific server var")
	format = flag.String("format", "", "What should we output")
	name   = flag.String("name", "", "Regex pattern for the name to lookup")
	group  = flag.String("group", "", "Regex pattern for the group to lookoup")
)

func main() {
	flag.Parse()
	serverpb, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	servers := []*pb.ServerFile_Server{}
	if len(*name) > 0 {
		got, err := findByName(serverpb, *name)
		if err != nil {
			fmt.Println(err)
		}
		servers = append(servers, got...)
	}

	if len(*group) > 0 {
		got, err := findByGroup(serverpb, *group)
		if err != nil {
			fmt.Println(err)
		}
		servers = append(servers, got...)
	}

	formatted := formatOutput(servers, *format)
	for _, f := range formatted {
		fmt.Println(f)
	}
}

// Read the text-format proto config file and unmarshal it
func loadConfig() (*pb.ServerFile, error) {
	cfg := &pb.ServerFile{}

	if *config == "" {
		homedir, err := os.UserHomeDir()
		sep := os.PathSeparator
		if err != nil {
			return nil, err
		}
		*config = fmt.Sprintf("%s%c%s", homedir, sep, serversFile)
	}

	raw, err := os.ReadFile(*config)
	if err != nil {
		return nil, err
	}

	err = prototext.Unmarshal(raw, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func findByName(src *pb.ServerFile, name string) ([]*pb.ServerFile_Server, error) {
	results := []*pb.ServerFile_Server{}
	reName, err := regexp.Compile(name)
	if err != nil {
		return results, err
	}

	for _, s := range src.GetServer() {
		if reName.MatchString(s.GetIdentifier()) {
			results = append(results, s)
		}
	}

	return results, nil
}

func findByGroup(src *pb.ServerFile, group string) ([]*pb.ServerFile_Server, error) {
	results := []*pb.ServerFile_Server{}
	reGroup, err := regexp.Compile(group)
	if err != nil {
		return results, err
	}

	for _, s := range src.GetServer() {
		for _, g := range s.GetGroups() {
			if reGroup.MatchString(g) {
				results = append(results, s)
			}
		}
	}

	return results, nil
}

func formatOutput(results []*pb.ServerFile_Server, format string) []string {
	final := []string{}
	for _, r := range results {
		formatted := format
		if strings.Contains(format, "%n") {
			formatted = strings.ReplaceAll(formatted, "%n", r.GetIdentifier())
		}
		if strings.Contains(format, "%a") {
			formatted = strings.ReplaceAll(formatted, "%a", r.GetAddress())
		}
		if strings.Contains(format, "%s") {
			formatted = strings.ReplaceAll(formatted, "%s", r.GetSshHost())
		}
		if strings.Contains(format, "%h") {
			tokens := strings.Split(r.GetAddress(), ":")
			if len(tokens) > 0 {
				formatted = strings.ReplaceAll(formatted, "%h", tokens[0])
			}
		}
		if strings.Contains(format, "%p") {
			tokens := strings.Split(r.GetAddress(), ":")
			if len(tokens) > 1 {
				formatted = strings.ReplaceAll(formatted, "%p", tokens[1])
			}
		}
		final = append(final, formatted)
	}

	return final
}
