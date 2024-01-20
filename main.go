// quake2 server resolver is a cli utility for pulling data from a text-format
// protobuf implimenting github.com/packetflinger/libq2/servers_file.proto
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	pb "github.com/packetflinger/libq2/proto"
	"google.golang.org/protobuf/encoding/prototext"
)

var (
	serversFile = ".q2servers.config" // default name, should be in home directory
	config      = flag.String("config", "", "Specify a server data file")
	format      = flag.String("format", "%n", "What should we output")
	name        = flag.String("name", "", "Regex pattern for the name to lookup")
	group       = flag.String("group", "", "Regex pattern for the group to lookoup")
	address     = flag.String("address", "", "Regex pattern for the address to lookup")
	union       = flag.Bool("union", false, "Combine results from multiple criteria")
)

func main() {
	if len(os.Args) < 2 {
		usage()
		return
	}

	flag.Parse()
	serverpb, err := loadConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	servers := []*pb.ServerFile_Server{}
	required := 0
	if len(*name) > 0 {
		got, err := findByName(serverpb, *name)
		if err != nil {
			fmt.Println(err)
		}
		servers = append(servers, got...)
		required++
	}

	if len(*group) > 0 {
		got, err := findByGroup(serverpb, *group)
		if err != nil {
			fmt.Println(err)
		}
		servers = append(servers, got...)
		required++
	}

	if len(*address) > 0 {
		got, err := findByAddress(serverpb, *address)
		if err != nil {
			fmt.Println(err)
		}
		servers = append(servers, got...)
		required++
	}

	if *union {
		servers = unique(servers)
	} else {
		servers = intersections(servers, required)
	}

	formatted := formatOutput(servers, *format)
	for _, f := range formatted {
		fmt.Println(f)
	}
}

func usage() {
	fmt.Printf("Usage: %s [flags]\n", os.Args[0])
	flag.PrintDefaults()
}

// Read the text-format proto config file and unmarshal it
func loadConfig() (*pb.ServerFile, error) {
	cfg := &pb.ServerFile{}

	if *config == "" {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		*config = path.Join(homedir, serversFile)
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

func findByAddress(src *pb.ServerFile, address string) ([]*pb.ServerFile_Server, error) {
	results := []*pb.ServerFile_Server{}
	reAddress, err := regexp.Compile(address)
	if err != nil {
		return results, err
	}

	for _, s := range src.GetServer() {
		if reAddress.MatchString(s.GetAddress()) {
			results = append(results, s)
		}
	}

	return results, nil
}

// Substitute all the placeholders in the format
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
		if strings.Contains(format, "%l") {
			formatted = strings.ReplaceAll(formatted, "%l", r.GetLogFile())
		}
		if strings.Contains(format, "%g") {
			formatted = strings.ReplaceAll(formatted, "%g", strings.Join(r.GetGroups(), ","))
		}
		final = append(final, formatted)
	}

	return final
}

// Return only the server objects that match ALL the criteria provided.
// required arg is the number of different criteria asked for, so servers
// that appear in the collection that many times are what we're looking for.
func intersections(s1 []*pb.ServerFile_Server, required int) []*pb.ServerFile_Server {
	var match []*pb.ServerFile_Server
	counts := make(map[string]int)
	if len(s1) == 0 {
		return match
	}

	for _, sv := range s1 {
		counts[sv.Identifier]++
	}

	for k, v := range counts {
		if v >= required {
			for _, sv := range s1 {
				if sv.GetIdentifier() == k {
					match = append(match, sv)
					break
				}
			}
		}
	}

	return match
}

// Remove any duplicates
func unique(s1 []*pb.ServerFile_Server) []*pb.ServerFile_Server {
	var newlist []*pb.ServerFile_Server
	seen := make(map[string]bool)
	for _, sv := range s1 {
		if _, ok := seen[sv.GetIdentifier()]; !ok {
			seen[sv.GetIdentifier()] = true
			newlist = append(newlist, sv)
		}
	}
	return newlist
}
