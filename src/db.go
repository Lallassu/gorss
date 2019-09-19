package main

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"strings"
	"time"
)

type DB struct {
	db *sql.DB
	c  *Controller
}

func (d *DB) Init(c *Controller) error {
	d.c = c
	db, err := sql.Open("sqlite3", "./gorss.db")
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
			deleted bool,
			published DATETIME
		);`)
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (d *DB) CleanupDB() {
	st, err := d.db.Prepare("delete from articles where published < date('now', '? day') and deleted = true")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err := st.Exec(d.c.conf.DaysToKeepDeletedArticlesInDB); err != nil {
		log.Println(err)
	}

	st2, err := d.db.Prepare("delete from articles where published < date('now', '? day') and read = true")
	if err != nil {
		log.Println(err)
	}
	defer st2.Close()

	if _, err := st2.Exec(d.c.conf.DaysToKeepReadArticlesInDB); err != nil {
		log.Println(err)
	}
}

func (d *DB) All() []Article {
	st, err := d.db.Prepare("select id,feed,title,content,published,link,read from articles where deleted = 0 order by id")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer st.Close()

	rows, err := st.Query()
	if err != nil {
		log.Println(err)
	}
	defer rows.Close()

	var (
		id        int
		title     string
		content   string
		feed      string
		link      string
		read      bool
		published time.Time
	)

	articles := []Article{}

	for rows.Next() {
		err = rows.Scan(&id, &feed, &title, &content, &published, &link, &read)
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
		articles = append(articles, Article{id: id, highlight: highlight, feed: feed, title: title, content: content, published: published, link: link, read: read})
	}
	return articles
}

func (d *DB) AllSaved() []Article {

	return nil
}

func (d *DB) Save(a Article) error {
	// First make sure that the same articl doesn't already exists.
	st, err := d.db.Prepare("select title from articles where title = ? order by id")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	res, err := st.Query(a.title)
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

	st, err = tx.Prepare("insert into articles(feed, title, content, link, read, published, deleted) values(?, ?, ?, ?, ?,?,?)")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err := st.Exec(a.feed, a.title, a.content, a.link, false, a.published, false); err != nil {
		log.Println(err)
	}

	tx.Commit()
	return nil
}

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

func (d *DB) MarkAllRead() {
	st, err := d.db.Prepare("update articles set read = true")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err := st.Exec(); err != nil {
		log.Println(err)
	}
}

func (d *DB) MarkAllUnread() {
	st, err := d.db.Prepare("update articles set read = false")
	if err != nil {
		log.Println(err)
	}
	defer st.Close()

	if _, err := st.Exec(); err != nil {
		log.Println(err)
	}
}
