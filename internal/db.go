package internal

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3" // nolint: golint
)

// DB holds the database information
type DB struct {
	db *sql.DB
	c  *Controller
}

// Init setups the database and creates tables if needed.
func (d *DB) Init(c *Controller, dbFile string) error {
	d.c = c
	db, err := sql.Open("sqlite3", dbFile)
	if err != nil {
		log.Println(err)
	}
	d.db = db
	//defer d.db.Close()

	_, err = d.db.Exec(`
         create table if not exists articles(
			id integer not null primary key,
			feed text,
			title text,
			content text,
			link text,
			read bool,
			display_name string,
			deleted bool,
			published DATETIME
		);`)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// CleanupDB removes old and deleted articles
func (d *DB) CleanupDB() {
	st, err := d.db.Prepare(fmt.Sprintf(
		"delete from articles where published < date('now', '-%d day') and deleted = true",
		d.c.conf.DaysToKeepDeletedArticlesInDB),
	)
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err = st.Exec(); err != nil {
		log.Println(err)
	}

	st2, err := d.db.Prepare(fmt.Sprintf(
		"delete from articles where published < date('now', '-%d day') and read = true",
		d.c.conf.DaysToKeepReadArticlesInDB),
	)
	if err != nil {
		log.Println(err)
	}
	defer st2.Close()

	if _, err := st2.Exec(); err != nil {
		log.Println(err)
	}
}

// All fetches all articles from the database
func (d *DB) All() []Article {
	st, err := d.db.Prepare("select id,feed,title,content,published,link,read,display_name from articles where deleted = false order by id")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer st.Close()

	rows, err := st.Query()
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var (
		id        int
		title     string
		content   string
		feed      string
		link      string
		read      bool
		display   string
		published time.Time
	)

	articles := []Article{}

	for rows.Next() {
		err = rows.Scan(&id, &feed, &title, &content, &published, &link, &read, &display)
		if err != nil {
			log.Println(err)
		}

		// Check if we should higlight it
		fields := strings.Fields(title)
		highlight := false
		for _, f := range fields {
			for _, h := range d.c.conf.Highlights {
				if strings.Contains(strings.ToLower(f), strings.ToLower(h)) {
					highlight = true
					break
				}
			}
			if highlight {
				break
			}
		}
		articles = append(articles, Article{id: id, highlight: highlight, feed: feed, title: title, content: content, published: published, link: link, read: read, feedDisplay: display})
	}
	return articles
}

// Save adds a new article to database if the title doesn't already exists.
func (d *DB) Save(a Article) error {
	// First make sure that the same article doesn't already exists.
	st, err := d.db.Prepare("select title from articles where feed = ? and title = ? order by id")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	res, err := st.Query(a.feed, a.title)
	if err != nil {
		log.Println(err)
	}
	defer res.Close()
	for res.Next() {
		return nil
	}

	tx, err := d.db.Begin()
	if err != nil {
		log.Println(err)
	}

	st, err = tx.Prepare("insert into articles(feed, title, content, link, read, display_name, published, deleted) values(?, ?, ?, ?, ?, ?, ?,?)")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err := st.Exec(a.feed, a.title, a.content, a.link, false, a.feedDisplay, a.published, false); err != nil {
		log.Println(err)
	}

	tx.Commit()
	return nil
}

// Delete marks an article as deleted. Will not remove it from DB (see CleanupDB)
func (d *DB) Delete(a *Article) {
	st, err := d.db.Prepare("update articles set deleted = true where id = ?")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err := st.Exec(a.id); err != nil {
		log.Println(err)
	}
}

// MarkRead marks an article as read in the database
func (d *DB) MarkRead(a *Article) error {
	st, err := d.db.Prepare("update articles set read = true where id = ?")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err := st.Exec(a.id); err != nil {
		log.Println(err)
	}
	return nil
}

// MarkUnread marks an article as unread in the database
func (d *DB) MarkUnread(a Article) error {
	st, err := d.db.Prepare("update articles set read = false where id = '?'")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if err := st.QueryRow(a.id); err != nil {
		log.Println(err)
	}
	return nil
}

// MarkAllRead marks all articles in the database as read
func (d *DB) MarkAllRead(feed string) {
	stmt := "update articles set read = true"
	if feed != "" {
		stmt = "update articles set read = true where feed = ?"
	}

	st, err := d.db.Prepare(stmt)
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if feed != "" {
		if _, err := st.Exec(feed); err != nil {
			log.Println(err)
		}
	} else {
		if _, err := st.Exec(); err != nil {
			log.Println(err)
		}
	}
}

// MarkAllUnread marks all articles in the database as not read
func (d *DB) MarkAllUnread(feed string) {
	stmt := "update articles set read = false"
	if feed != "" {
		stmt = "update articles set read = false where feed = ?"
	}

	st, err := d.db.Prepare(stmt)
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if feed != "" {
		if _, err := st.Exec(feed); err != nil {
			log.Println(err)
		}
	} else {
		if _, err := st.Exec(); err != nil {
			log.Println(err)
		}
	}
}
