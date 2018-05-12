package main

import (
	"io/ioutil"
	"os"
	"testing"
)

// Test basic primitives.
func TestReadWrite(t *testing.T) {

	//
	// Create a temporary directory
	//
	p, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err == nil {
		PREFIX = p + "/"
	} else {
		t.Fatal(err)
	}

	//
	// Reading a file will fail
	//
	key := "this-is-a-key"
	out, err := readFile(key)

	if err == nil {
		t.Errorf("No error reading a missing file, when we expected one")
	}

	//
	// Now write some content
	//
	err = writeFile(key, key)
	if err != nil {
		t.Errorf("Error writing to a test file")
	}

	//
	// At this point reading it back should succeed
	//
	out, err = readFile(key)

	if err != nil {
		t.Errorf("Error reading file - we didn't expect that")
	}
	if out != key {
		t.Errorf("Got unexpected result reading file")
	}

	//
	// Because of the way we work reading the KeyFromAuth will
	// have the same result
	//
	out2, err2 := KeyFromAuth(key)
	if out2 != out {
		t.Errorf("Data mismatch")
	}
	if err2 != nil {
		t.Errorf("Error reading data again")
	}
	//
	// Cleanup our temporary directory
	//
	os.RemoveAll(p)
}

//
// Test fetching a markdown file, and editing it.
//
func TestReadEditMarkdown(t *testing.T) {

	//
	// Create a temporary directory
	//
	p, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err == nil {
		PREFIX = p + "/"
	} else {
		t.Fatal(err)
	}

	//
	// Write some content
	//
	key := "this-is-a-key"
	err = writeFile(key+".TEXT", "_Italic_")
	if err != nil {
		t.Errorf("Error writing 'markdown'")
	}

	//
	// Get the "markdown"
	//
	var data string
	data = getMarkdown(key)

	if data != "_Italic_" {
		t.Errorf("Got unexpected result reading file")
	}

	//
	// Update it
	//
	err = UpdateMarkdown(key, "__Bold__")
	if err != nil {
		t.Errorf("Error updating markdown")
	}

	//
	// Read it again
	//
	data = getMarkdown(key)

	if data != "__Bold__" {
		t.Errorf("Got unexpected result reading file")
	}
	//
	// Cleanup our temporary directory
	//
	os.RemoveAll(p)
}

//
// Test deleting a markdown file
//
func TestDeleteMarkdown(t *testing.T) {

	//
	// Create a temporary directory
	//
	p, err := ioutil.TempDir(os.TempDir(), "prefix")
	if err == nil {
		PREFIX = p + "/"
	} else {
		t.Fatal(err)
	}

	//
	// Save some markdown
	//
	key, auth, err := SaveMarkdown("__bold__", "127.127.53.53")
	if err != nil {
		t.Errorf("Error saving markdown")
	}

	//
	// Get the markdown, to ensure it worked
	//
	var data string
	data = getMarkdown(key)
	if data != "__bold__" {
		t.Errorf("Got unexpected result reading file")
	}

	//
	// Now delete
	//
	err = DeleteMarkdown(auth)
	if err != nil {
		t.Errorf("Unexpected error deleting: %s", err.Error())
	}

	//
	// Get the markdown, again
	//
	data = getMarkdown(key)
	if data != "" {
		t.Errorf("Got unexpected result reading file")
	}

	//
	// Cleanup our temporary directory
	//
	os.RemoveAll(p)
}
