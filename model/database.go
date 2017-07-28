package model

import (
	"database/sql"

	wp "github.com/SherClockHolmes/webpush-go"
	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func OpenDatabase(path string) (*Database, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	database := &Database{
		db: db,
	}
	err = database.init()
	if err != nil {
		return nil, err
	}
	return database, nil
}

func (db *Database) init() error {
	create := `create table if not exists subscriptions (endpoint text, key blob, auth blob, unique (endpoint, key, auth) on conflict replace);`
	_, err := db.db.Exec(create)
	return err
}

func (db *Database) Close() {
	db.db.Close()
}

func (db *Database) Subscribe(sub *wp.Subscription) error {
	transaction, err := db.db.Begin()
	if err != nil {
		return err
	}

	statement, err := transaction.Prepare("insert into subscriptions(endpoint, key, auth) values(?, ?, ?);")
	if err != nil {
		return err
	}
	defer statement.Close()

	_, err = statement.Exec(sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth)
	if err != nil {
		return err
	}
	return transaction.Commit()
}

func (db *Database) GetSubscriptions() ([]*wp.Subscription, error) {
	rows, err := db.db.Query("select endpoint, key, auth from subscriptions;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var subs []*wp.Subscription
	for rows.Next() {
		sub := &wp.Subscription{}
		err = rows.Scan(&sub.Endpoint, &sub.Keys.P256dh, &sub.Keys.Auth)
		if err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, nil
}
