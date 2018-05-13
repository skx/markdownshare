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

type IPCmd struct {
	source string
}

//
// Glue
//
func (*IPCmd) Name() string     { return "ips" }
func (*IPCmd) Synopsis() string { return "Show counts of uploads by each distinct IP addresses." }
func (*IPCmd) Usage() string {
	return `ips :
  Show the number of uploads each IP has made
`
}

//
// Flag setup
//
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

//
// Entry-point.
//
func (p *IPCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	if p.source != "" {
		showIP(p.source)
	} else {
		showStats()
	}
	return subcommands.ExitSuccess
}
