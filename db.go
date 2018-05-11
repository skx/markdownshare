//
// This package contains our SQLite DB interface.  It is a little ropy.
//

package main

import (
	"database/sql"
	"errors"

	_ "github.com/mattn/go-sqlite3"
	"github.com/satori/go.uuid"
)

// getMarkdown returns the markdown from the given public-key.
func getMarkdown(key string) (string, error) {
	var db *sql.DB
	var err error

	//
	// Return if the database already exists.
	//
	db, err = sql.Open("sqlite3", "predis.db")
	if err != nil {
		return "", err
	}
	defer db.Close()

	var out string
	row := db.QueryRow("SELECT val FROM string WHERE key=?",
		"MARKDOWN:"+key+":TEXT")
	err = row.Scan(&out)

	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return "", errors.New("Markdown not found")
	default:
	}

	return out, nil
}

// KeyFromAuth returns the public-key from the (private) auth-token
// it is used for Delete/Edit operations
func KeyFromAuth(auth string) (string, error) {
	db, err := sql.Open("sqlite3", "predis.db")
	if err != nil {
		return "", err
	}
	defer db.Close()

	//
	// Find the public-ID which corresponds to the auth-key
	//
	var key string
	row := db.QueryRow("SELECT val FROM string WHERE key=?", "MARKDOWN:KEY:"+auth)
	err = row.Scan(&key)

	switch {
	case err == sql.ErrNoRows:
	case err != nil:
		return "", errors.New("Invalid authentication token")
	}
	return key, nil

}

// UpdateMarkdown updates the text associated with the given
// key - and is only called as a result of an edit operation.
func UpdateMarkdown(key string, markdown string) error {

	db, err := sql.Open("sqlite3", "predis.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// The query we run
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	up, err := tx.Prepare("UPDATE string SET val=? WHERE key=?")
	if err != nil {
		return err
	}

	_, err = up.Exec(markdown, "MARKDOWN:"+key+":TEXT")
	if err != nil {
		return err
	}
	tx.Commit()

	return nil
}

// SaveMarkdown adds a new database entry, recording the text
// and the IP address of the submitter.
func SaveMarkdown(markdown string, ip string) (string, string, error) {

	db, err := sql.Open("sqlite3", "predis.db")
	if err != nil {
		return "", "", err
	}
	defer db.Close()

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

	tx, err := db.Begin()
	if err != nil {
		return "", "", err
	}

	stmt, err := tx.Prepare("INSERT INTO string (key,val) VALUES(?,?)")
	if err != nil {
		return "", "", err
	}
	defer stmt.Close()

	stmt.Exec("MARKDOWN:"+key+":TEXT", markdown)
	stmt.Exec("MARKDOWN:"+key+":IP", ip)
	stmt.Exec("MARKDOWN:"+key+":AUTH", auth)
	stmt.Exec("MARKDOWN:KEY:"+auth, key)

	tx.Commit()
	return key, auth, nil
}

// DeleteMarkdown deletes all the data associated with a given paste,
// via the authentication-key supplied.
func DeleteMarkdown(auth string) error {

	key, err := KeyFromAuth(auth)
	if err != nil {
		return err
	}

	var db *sql.DB
	db, err = sql.Open("sqlite3", "predis.db")
	if err != nil {
		return err
	}
	defer db.Close()

	//
	// OK now we've got a list of things to delete
	//
	queries := []string{
		"DELETE FROM string where key='MARKDOWN:" + key + ":TEXT'",
		"DELETE FROM string where key='MARKDOWN:" + key + ":IP'",
		"DELETE FROM string where key='MARKDOWN:" + key + ":AUTH'",
		"DELETE FROM string where key='MARKDOWN:KEY:" + auth + "'",
	}

	for _, ent := range queries {
		_, err = db.Exec(ent)
		if err != nil {
			return err
		}
	}

	return nil
}
