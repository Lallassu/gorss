package main

import (
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"log"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Controller handles the logic and keep everything together
type Controller struct {
	rss         *RSS
	db          *DB
	win         *Window
	activeFeed  string
	linksToOpen []string
	quit        chan int
	articles    []Article
	aLock       sync.Mutex
	conf        Config
	theme       Theme
	prevArticle *Article
	lastUpdate  time.Time
}

// Init initiates the controller with database handles etc.
// It also starts the update loop and window handling.
func (c *Controller) Init(cfg, theme string) {
	c.quit = make(chan int)

	c.conf = LoadConfiguration(cfg)
	c.theme = LoadTheme(theme)

	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	c.articles = make([]Article, 0)

	c.db = &DB{}
	if err := c.db.Init(c); err != nil {
		log.Fatal("Database init failed.")
	}

	c.win = &Window{}
	c.win.Init(c.Input, c)

	c.rss = &RSS{}
	c.rss.Init(c)

	c.win.RegisterSelectedFunc(c.SelectArticle)
	c.win.RegisterSelectionChangedFunc(c.SelectArticle)
	c.win.RegisterSelectedFeedFunc(c.SelectFeed)

	c.GetArticlesFromDB()
	c.ShowFeeds()

	c.UpdateLoop()

	c.win.Start()
}

// GetConfigKeys creates a list of keys that we want to present
// in the help section.
func (c *Controller) GetConfigKeys() map[string]string {
	keys := make(map[string]string, 0)

	keys["Open Link"] = c.conf.KeyOpenLink
	keys["Mark Link"] = c.conf.KeyMarkLink
	keys["Open Marked"] = c.conf.KeyOpenMarked
	keys["Delete"] = c.conf.KeyDeleteArticle
	keys["Up"] = c.conf.KeyMoveUp
	keys["Down"] = c.conf.KeyMoveDown
	keys["Sort by date"] = c.conf.KeySortByDate
	keys["Sort by feed"] = c.conf.KeySortByFeed
	keys["Sort by title"] = c.conf.KeySortByTitle
	keys["Sort by unread"] = c.conf.KeySortByUnread
	keys["Toggle Preview"] = c.conf.KeyTogglePreview
	keys["Mark All Read"] = c.conf.KeyMarkAllRead
	keys["Mark All UnRead"] = c.conf.KeyMarkAllUnread
	keys["Toggle Help"] = "h"
	keys["Select Feed Window"] = c.conf.KeySelectFeedWindow
	keys["Select Article Window"] = c.conf.KeySelectArticleWindow
	keys["Select Preview Window"] = c.conf.KeySelectPreviewWindow
	for _, cmd := range c.conf.CustomCommands {
		keys[cmd.Cmd] = cmd.Key
	}

	return keys
}

// UpdateLoop updates the feeds and windows
func (c *Controller) UpdateLoop() {
	c.UpdateFeeds() // Start by updating feeds.
	c.GetArticlesFromDB()
	c.ShowFeeds()
	go func() {
		updateWin := time.NewTicker(time.Duration(5) * time.Second)
		updateFeeds := time.NewTicker(time.Duration(c.conf.SecondsBetweenUpdates) * time.Second)
		for {
			select {
			case <-updateWin.C:
				// Don't update unread articles since it will remove the current article.
				if c.activeFeed != "unread" {
					c.ShowArticles(c.activeFeed) // Update article list (timestamps etc are then updated)
				}
				c.ShowFeeds()
			case <-updateFeeds.C:
				c.UpdateFeeds()
				c.db.CleanupDB()
				c.win.StatusUpdate()
			case <-c.quit:
				c.Quit()
				return
			}
		}
	}()
}

// Quit ends the application
func (c *Controller) Quit() {
	c.win.app.Stop()
	os.Exit(0)
}

// UpdateFeeds updates the articles kept in the controller
func (c *Controller) UpdateFeeds() {
	c.rss.Update()
	for _, f := range c.rss.feeds {
		if f == nil {
			continue
		}
		for _, item := range f.Items {
			if item == nil {
				continue
			}

			published := time.Now()
			if item.PublishedParsed != nil {
				published = *item.PublishedParsed
			} else {
				published = *item.UpdatedParsed
			}
			content := item.Description
			if content == "" {
				content = item.Content
			}
			a := Article{
				c:         c,
				feed:      f.Title,
				title:     item.Title,
				content:   content,
				link:      item.Link,
				published: published,
				read:      false,
			}
			// Make sure the same article doesn't exists.
			exists := false
			for _, e := range c.articles {
				if e.title == a.title {
					exists = true
				}
			}

			if !exists {
				c.db.Save(a)
			}
		}
	}
	c.lastUpdate = time.Now()
	c.GetArticlesFromDB()
	c.ShowArticles(c.activeFeed)
}

// GetArticlesFromDB fetches all articles from the database
func (c *Controller) GetArticlesFromDB() {
	c.articles = []Article{}
	articles := c.db.All()
	for _, a := range articles {
		a.c = c
		c.articles = append(c.articles, a)
	}
}

// OpenLink opens a link in the default webbrowser.
func (c *Controller) OpenLink(link string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", link).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", link).Start()
	case "darwin":
		err = exec.Command("open", link).Start()
	}
	if err != nil {
		log.Println(err)
	}
}

// ShowFeeds updates the feeds window with the current feeds and statuses
func (c *Controller) ShowFeeds() {
	c.win.ClearFeeds()

	hc := 0
	total := 0
	for _, a := range c.articles {
		if a.highlight {
			total++
		}
		if a.highlight && !a.read {
			hc++
		}
	}
	c.win.AddToFeeds(fmt.Sprintf("[%s]Highlight", c.theme.Highlights), hc, total, &Article{feed: "highlight"})

	feeds := make(map[string]int, 0)
	feedsTotal := make(map[string]int, 0)
	urTotal := 0
	total = 0
	for _, a := range c.articles {
		total++
		if _, ok := feeds[a.feed]; !ok {
			feeds[a.feed] = 0
			feedsTotal[a.feed] = 0
		}
		feedsTotal[a.feed]++
		if !a.read {
			feeds[a.feed]++
			urTotal++
		}
	}

	c.win.AddToFeeds("Unread", urTotal, urTotal, &Article{feed: "unread"})
	c.win.AddToFeeds("All Articles", total, urTotal, &Article{feed: "allarticles"})

	// for k, v := range feeds {
	// 	c.win.AddToFeeds(k, v, feedsTotal[k], &Article{feed: k})
	// }
	var keys []string
	for k := range feeds {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// To perform the opertion you want
	for _, k := range keys {
		//fmt.Println("Key:", k, "Value:", m[k])
		c.win.AddToFeeds(k, feeds[k], feedsTotal[k], &Article{feed: k})
	}
}

// ShowArticles lists all articles for the current feed
func (c *Controller) ShowArticles(feed string) {
	c.aLock.Lock()
	defer c.aLock.Unlock()
	c.win.ClearArticles()

	if feed == "" {
		feed = "highlight"
	}

	c.activeFeed = feed

	for i, a := range c.articles {
		if feed == "highlight" {
			if !a.highlight {
				continue
			}
		} else if feed == "allarticles" {
			// pass - take all articles
		} else if feed == "unread" {
			if a.read {
				continue
			}
		} else {
			if a.feed != c.activeFeed {
				continue
			}
		}
		markedWeb := false

		for _, s := range c.linksToOpen {
			if s == a.link {
				markedWeb = true
				break
			}
		}
		c.win.AddToArticles(&c.articles[i], markedWeb)
	}
	c.win.articles.ScrollToBeginning()
}

// GetArticleForSelection returns the article instance for the selected article
// in the aritcles table.
func (c *Controller) GetArticleForSelection() *Article {
	if !c.win.ArticlesHasFocus() {
		return nil
	}

	var cell *tview.TableCell

	if c.activeFeed == "unread" {
		cell = c.win.articles.GetCell(1, 3)
	} else {
		r, _ := c.win.articles.GetSelection()
		cell = c.win.articles.GetCell(r, 3)
	}
	ref := cell.GetReference()
	if ref != nil {
		return ref.(*Article)
	}
	return nil
}

// SelectFeed
func (c *Controller) SelectFeed(row, col int) {
	if row <= 0 {
		return
	}
	sort.Slice(c.articles, func(i, j int) bool {
		return c.articles[i].published.String() > c.articles[j].published.String()
	})
	r, _ := c.win.feeds.GetSelection()
	cell := c.win.feeds.GetCell(r, 2)
	ref := cell.GetReference()
	if ref != nil {
		c.ShowArticles(ref.(*Article).feed)
	}
}

func (c *Controller) SelectArticle(row, col int) {
	if c.activeFeed == "unread" && row == 0 {
		if c.prevArticle != nil {
			c.db.MarkRead(c.prevArticle)
			c.prevArticle.read = true
			c.ShowArticles(c.activeFeed)
			c.ShowFeeds()
		}
		return
	}
	a := c.GetArticleForSelection()
	if a == nil {
		return
	}

	c.win.preview.Clear()

	if c.activeFeed != "unread" {
		c.db.MarkRead(a)
		a.read = true
	}
	c.prevArticle = a

	c.win.AddPreview(a)

	c.ShowArticles(c.activeFeed)
	c.ShowFeeds()

	if c.prevArticle != nil {
		c.db.MarkRead(c.prevArticle)
		c.prevArticle.read = true
	}

}

func (c *Controller) Input(e *tcell.EventKey) *tcell.EventKey {
	keyName := string(e.Name())
	if strings.Contains(keyName, "Rune") {
		keyName = string(e.Rune())
	}

	if keyName == c.conf.KeyQuit {
		c.quit <- 1
		return nil
	}

	if keyName == c.conf.KeySwitchWindows {
		c.win.SwitchFocus()
		return nil
	}

	if keyName == c.conf.KeyMarkLink {
		a := c.GetArticleForSelection()
		if a == nil {
			return nil
		}
		// Check if already added. Then unadd it.
		removed := false
		for i, l := range c.linksToOpen {
			if l == a.link {
				c.linksToOpen[i] = c.linksToOpen[len(c.linksToOpen)-1]
				c.linksToOpen[len(c.linksToOpen)-1] = ""
				c.linksToOpen = c.linksToOpen[:len(c.linksToOpen)-1]
				removed = true
				break
			}
		}
		if !removed {
			c.linksToOpen = append(c.linksToOpen, a.link)
		}
		c.ShowArticles(c.activeFeed)
		return nil
	}

	// Open selected in browser
	if keyName == c.conf.KeyOpenLink {
		a := c.GetArticleForSelection()
		if a == nil {
			return nil
		}
		c.OpenLink(a.link)
		return nil
	}

	// Delete article from DB and list
	if keyName == c.conf.KeyDeleteArticle {
		a := c.GetArticleForSelection()
		if a != nil {
			c.db.Delete(a)
			// Also delete in linksToOpen list
			for i, l := range c.linksToOpen {
				if l == a.link {
					c.linksToOpen = append(c.linksToOpen[:i], c.linksToOpen[i+1:]...)
					break
				}
			}

			for i, ca := range c.articles {
				if ca.id == a.id {
					c.articles = append(c.articles[:i], c.articles[i+1:]...)
					break
				}
			}
			c.ShowArticles(c.activeFeed)
			c.ShowFeeds()
		}
		return nil
	}

	if keyName == c.conf.KeyMoveDown {
		if c.activeFeed == "unread" {
			c.win.articles.Select(0, 3)
		}
		c.win.MoveDown()
		return nil
	}

	if keyName == c.conf.KeyMoveUp {
		c.win.MoveUp()
		return nil
	}

	// Sort on feed
	if keyName == c.conf.KeySortByFeed {
		sort.Slice(c.articles, func(i, j int) bool {
			return c.articles[i].feed < c.articles[j].feed
		})
		c.ShowArticles(c.activeFeed)
		return nil
	}

	// Sort on title
	if keyName == c.conf.KeySortByTitle {
		sort.Slice(c.articles, func(i, j int) bool {
			return c.articles[i].title < c.articles[j].title
		})
		c.ShowArticles(c.activeFeed)
		return nil
	}
	// Sort on date
	if keyName == c.conf.KeySortByDate {
		sort.Slice(c.articles, func(i, j int) bool {
			return c.articles[i].published.String() > c.articles[j].published.String()
		})
		c.ShowArticles(c.activeFeed)
		return nil
	}

	// Sort on unread
	if keyName == c.conf.KeySortByUnread {
		sort.Slice(c.articles, func(i, j int) bool {
			return strconv.FormatBool(c.articles[i].read) < strconv.FormatBool(c.articles[j].read)
		})
		c.ShowArticles(c.activeFeed)
		return nil
	}

	// Mark all read
	if keyName == c.conf.KeyMarkAllRead {
		c.db.MarkAllRead()
		c.GetArticlesFromDB()
		c.ShowArticles(c.activeFeed)
		c.ShowFeeds()
		return nil
	}

	// Mark all unread
	if keyName == c.conf.KeyMarkAllUnread {
		c.db.MarkAllUnread()
		c.GetArticlesFromDB()
		c.ShowArticles(c.activeFeed)
		c.ShowFeeds()
		return nil
	}

	// Switch to feeds
	if keyName == c.conf.KeySelectFeedWindow {
		c.win.SelectFeedWindow()
		return nil
	}
	if keyName == c.conf.KeySelectArticleWindow {
		c.win.SelectArticleWindow()
		return nil
	}
	if keyName == c.conf.KeySelectPreviewWindow {
		c.win.SelectPreviewWindow()
		return nil
	}

	// Open all marked links
	if keyName == c.conf.KeyOpenMarked {
		for _, l := range c.linksToOpen {
			c.OpenLink(l)
		}
		c.linksToOpen = []string{}
		c.ShowArticles(c.activeFeed)
		return nil
	}

	// Toggle preview
	if keyName == c.conf.KeyTogglePreview {
		c.win.TogglePreview()
		return nil
	}

	// Update feeds
	if keyName == c.conf.KeyUpdateFeeds {
		c.UpdateFeeds()
		return nil
	}

	// Toggle help
	if keyName == c.conf.KeyToggleHelp {
		c.win.ToggleHelp()
		return nil
	}

	for _, cmd := range c.conf.CustomCommands {
		if keyName == cmd.Key {
			// Substitute the parts we have support for
			a := c.GetArticleForSelection()
			if a != nil {
				cmdStr := cmd.Cmd
				cmdStr = strings.ReplaceAll(cmdStr, "ARTICLE.Title", a.title)
				cmdStr = strings.ReplaceAll(cmdStr, "ARTICLE.Link", a.link)
				cmdStr = strings.ReplaceAll(cmdStr, "ARTICLE.Feed", a.feed)
				cmdStr = strings.ReplaceAll(cmdStr, "ARTICLE.Content", a.content)

				command := exec.Command("/bin/sh", "-c", cmdStr)
				if err := command.Run(); err != nil {
					log.Printf("Failed to run command: %v", cmdStr)
				}
				return nil
			}
		}
	}

	// Fallback to default only if none above matches
	return e
}
