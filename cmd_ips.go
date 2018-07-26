//
// Show how many uploads each IP has made.
//

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/subcommands"
)

// IPCmd holds the options set by our command-line flags.
type IPCmd struct {
	source string
}

//
// Glue
//

// Name returns our command-name
func (*IPCmd) Name() string { return "ips" }

// Synopsis returns our command-synopsis
func (*IPCmd) Synopsis() string { return "Show counts of uploads by each distinct IP addresses." }

// Usage returns our usage information.
func (*IPCmd) Usage() string {
	return `ips :
  Show the number of uploads each IP has made
`
}

// SetFlags updates the IPCmd structure with command-line flags.
func (p *IPCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.source, "source", "", "Show uploads from the given source IP.")
}

// SHow the IDs of uploads by the given IP.
func showIP(ip string) {
	//
	// Get the list of files beneath our tree
	//
	fileList := []string{}
	_ = filepath.Walk(PREFIX, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".IP") {
			fileList = append(fileList, path)
		}
		return nil
	})

	//
	// For each one ..
	//
	for _, filename := range fileList {

		//
		// Read the file contents.
		//
		data, err := ioutil.ReadFile(filename)

		if err == nil {
			sip := string(data)
			sip = strings.TrimSpace(string(sip))

			if sip == ip {
				id := path.Base(filename)
				id = strings.TrimSuffix(id, ".IP")
				fmt.Printf("https://markdownshare.com/view/%s\n", id)
			}
		}
	}

}

// showStats shows the IPs which have uploaded content
func showStats() {
	//
	// Map for our results
	//
	data := make(map[string]int)

	//
	// Get the list of files beneath our tree
	//
	fileList := []string{}
	_ = filepath.Walk(PREFIX, func(path string, f os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".IP") {
			fileList = append(fileList, path)
		}
		return nil
	})

	//
	// For each one ..
	//
	for _, filename := range fileList {

		//
		// Read the file contents.
		//
		ip, err := ioutil.ReadFile(filename)

		if err == nil {
			sip := string(ip)
			sip = strings.TrimSpace(string(ip))
			data[sip] += 1
		}
	}

	//
	// Now show the results - sorted by key, via this
	// intermediary step.
	//
	type kv struct {
		Key   string
		Value int
	}

	var ss []kv
	for k, v := range data {
		ss = append(ss, kv{k, v})
	}

	sort.Slice(ss, func(i, j int) bool {
		return ss[i].Value > ss[j].Value
	})

	for _, kv := range ss {
		fmt.Printf("%s %d\n", kv.Key, kv.Value)
	}

}

// Execute is our entry-point to this sub-command
func (p *IPCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	if p.source != "" {
		showIP(p.source)
	} else {
		showStats()
	}
	return subcommands.ExitSuccess
}
