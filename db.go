//
// This package contains our "DB interface"
//
// In truth we don't use a database, instead we write to files upon
// the filesystem, beneath a given prefix.
//
// This is easier to backup, inspect, and move.
//

package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/satori/go.uuid"
)

var PREFIX = "store/"

func filePath(key string) string {

	res := PREFIX
	res += string(key[0]) + string("/")
	res += string(key[1]) + "/"
	res += string(key[2]) + "/"
	res += string(key[3]) + "/"

	// Make this when before we read-write
	os.MkdirAll(res, 0755)

	res += key

	return res
}

func readFile(key string) (string, error) {
	res, err := ioutil.ReadFile(filePath(key))
	if err != nil {
		return "", errors.New("Markdown not found")
	}
	return string(res), err
}

func writeFile(key string, data string) error {
	err := ioutil.WriteFile(filePath(key), []byte(data), 0644)
	if err != nil {
		fmt.Printf("ERROR WRITING: %s\n", err.Error())
	}
	return err
}

// getMarkdown returns the markdown from the given public-key.
func getMarkdown(key string) (string, error) {
	out, _ := readFile(key + ".TEXT")
	return out, nil
}

// KeyFromAuth returns the public-key from the (private) auth-token
// it is used for Delete/Edit operations
func KeyFromAuth(auth string) (string, error) {
	return (readFile(auth))

}

// UpdateMarkdown updates the text associated with the given
// key - and is only called as a result of an edit operation.
func UpdateMarkdown(key string, markdown string) error {
	return (writeFile(key+".TEXT", markdown))
}

// SaveMarkdown adds a new database entry, recording the text
// and the IP address of the submitter.
func SaveMarkdown(markdown string, ip string) (string, string, error) {

	// Generate the auth cookie
	var auth string
	var key string

	u1, err := uuid.NewV4()
	if err != nil {
		return "", "", err
	}
	auth = u1.String()

	u2, err := uuid.NewV4()
	if err != nil {
		return "", "", err
	}
	key = u2.String()

	writeFile(key+".TEXT", markdown)
	writeFile(key+".IP", ip)
	writeFile(auth, key)

	return key, auth, nil
}

// DeleteMarkdown deletes all the data associated with a given paste,
// via the authentication-key supplied.
func DeleteMarkdown(auth string) error {

	key, err := KeyFromAuth(auth)
	if err != nil {
		return err
	}

	//
	// These are the files we should delete
	//
	queries := []string{
		filePath(key) + ".TEXT",
		filePath(key) + ".IP",
		filePath(key) + ".AUTH",
		filePath(auth),
	}

	for _, ent := range queries {
		os.Remove(ent)
	}

	return nil
}
