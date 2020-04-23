package internal

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
	isUpdated   bool
	prevArticle *Article
	undoArticle *Article
	lastUpdate  time.Time
}

// Init initiates the controller with database handles etc.
// It also starts the update loop and window handling.
func (c *Controller) Init(cfg, theme, db string) {
	c.quit = make(chan int)

	c.conf = LoadConfiguration(cfg)
	c.theme = LoadTheme(theme)

	c.articles = make([]Article, 0)

	c.db = &DB{}
	if err := c.db.Init(c, db); err != nil {
		log.Fatal("Database init failed.")
	}

	c.win = &Window{}
	c.win.Init(c.Input, c)

	c.rss = &RSS{}
	c.rss.Init(c)

	c.win.RegisterSelectedFunc(c.SelectArticle)
	c.win.RegisterSelectionChangedFunc(c.SelectArticle)
	c.win.RegisterSelectedFeedFunc(c.SelectFeed)

	c.db.CleanupDB()

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
	keys["Update Feeds"] = c.conf.KeyUpdateFeeds
	keys["Switch Windows"] = c.conf.KeySwitchWindows
	keys["Quit"] = c.conf.KeyQuit

	for _, cmd := range c.conf.CustomCommands {
		keys[cmd.Cmd] = cmd.Key
	}

	return keys
}

// UpdateLoop updates the feeds and windows
func (c *Controller) UpdateLoop() {
	c.GetArticlesFromDB()
	c.UpdateFeeds() // Start by updating feeds.
	c.ShowFeeds()
	go func() {
		updateWin := time.NewTicker(time.Duration(30) * time.Second)
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

			var published time.Time
			if item.PublishedParsed != nil {
				published = *item.PublishedParsed
			} else if item.UpdatedParsed != nil {
				published = *item.UpdatedParsed
			} else {
				published = time.Now()
			}

			// Don't include old aritcles
			if int(time.Now().Sub(published).Hours()/24) > c.conf.SkipArticlesOlderThanDays {
				continue
			}

			// Transform the timestamp to local time

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
	c.isUpdated = true

	// On update, sort by date.
	sort.Slice(c.articles, func(i, j int) bool {
		return c.articles[i].published.String() > c.articles[j].published.String()
	})

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

	// If there are no unread left, then we remove the prevArticle so that
	// we don't add it again when updating the window.
	if urTotal == 0 {
		c.prevArticle = nil
	}

	c.win.AddToFeeds("All Articles", urTotal, total, &Article{feed: "allarticles"})

	var keys []string
	for k := range feeds {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
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
			if c.prevArticle != nil && c.isUpdated {
				if c.prevArticle.id != a.id && a.read {
					continue
				}
			} else {
				if a.read {
					continue
				}
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
	c.isUpdated = false

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

// SelectFeed is used as a callback for feed selection
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

// SelectArticle is used a hook for article selection
func (c *Controller) SelectArticle(row, col int) {
	if c.activeFeed == "unread" && row == 0 {
		if c.prevArticle != nil {
			c.db.MarkRead(c.prevArticle)
			c.prevArticle.read = true
			c.ShowArticles(c.activeFeed)
			c.ShowFeeds()
			c.win.ClearPreview()
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
	c.undoArticle = c.prevArticle
	c.prevArticle = a

	c.win.AddPreview(a)

	c.ShowArticles(c.activeFeed)
	c.ShowFeeds()

	if c.prevArticle != nil {
		c.db.MarkRead(c.prevArticle)
		c.prevArticle.read = true
	}

}

// Input handles keystrokes
func (c *Controller) Input(e *tcell.EventKey) *tcell.EventKey {
	keyName := string(e.Name())
	if strings.Contains(keyName, "Rune") {
		keyName = string(e.Rune())
	}

	switch keyName {
	case c.conf.KeyQuit:
		c.quit <- 1

	case c.conf.KeySwitchWindows:
		c.win.SwitchFocus()

	case c.conf.KeyMarkLink:
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
			// Append the linkmarker icon
			r, _ := c.win.articles.GetSelection()
			cell := c.win.articles.GetCell(r, 1)
			cell.SetText(fmt.Sprintf("%s%s", c.theme.LinkMarker, c.theme.UnreadMarker))
		} else {
			// Remove the linkmarker icon
			r, _ := c.win.articles.GetSelection()
			cell := c.win.articles.GetCell(r, 1)
			cell.SetText(fmt.Sprintf("%s", c.theme.UnreadMarker))
		}
		if c.activeFeed != "unread" {
			c.ShowArticles(c.activeFeed)
		}

	case c.conf.KeyOpenLink:
		a := c.GetArticleForSelection()
		if a == nil {
			return nil
		}
		c.OpenLink(a.link)

	case c.conf.KeyDeleteArticle:
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

	case c.conf.KeyMoveDown, "Down":
		if c.activeFeed == "unread" {
			c.win.articles.Select(0, 3)
		}
		c.win.MoveDown()

	case c.conf.KeyMoveUp, "Up":
		c.win.MoveUp()

	case c.conf.KeySortByFeed:
		sort.Slice(c.articles, func(i, j int) bool {
			return c.articles[i].feed < c.articles[j].feed
		})
		c.ShowArticles(c.activeFeed)

	case c.conf.KeySortByTitle:
		sort.Slice(c.articles, func(i, j int) bool {
			return c.articles[i].title < c.articles[j].title
		})
		c.ShowArticles(c.activeFeed)

	case c.conf.KeySortByDate:
		sort.Slice(c.articles, func(i, j int) bool {
			return c.articles[i].published.String() > c.articles[j].published.String()
		})
		c.ShowArticles(c.activeFeed)

	case c.conf.KeySortByUnread:
		sort.Slice(c.articles, func(i, j int) bool {
			return strconv.FormatBool(c.articles[i].read) < strconv.FormatBool(c.articles[j].read)
		})
		c.ShowArticles(c.activeFeed)

	case c.conf.KeyMarkAllRead:
		c.db.MarkAllRead()
		c.GetArticlesFromDB()
		c.ShowArticles(c.activeFeed)
		c.ShowFeeds()

	case c.conf.KeyMarkAllUnread:
		c.db.MarkAllUnread()
		c.GetArticlesFromDB()
		c.ShowArticles(c.activeFeed)
		c.ShowFeeds()

	case c.conf.KeySelectFeedWindow:
		c.win.SelectFeedWindow()

	case c.conf.KeySelectArticleWindow:
		c.win.SelectArticleWindow()

	case c.conf.KeySelectPreviewWindow:
		c.win.SelectPreviewWindow()

	case c.conf.KeyOpenMarked:
		for _, l := range c.linksToOpen {
			c.OpenLink(l)
		}
		c.linksToOpen = []string{}
		c.ShowArticles(c.activeFeed)

	case c.conf.KeyTogglePreview:
		c.win.TogglePreview()

	case c.conf.KeyUpdateFeeds:
		c.UpdateFeeds()

	case c.conf.KeyToggleHelp:
		c.win.ToggleHelp()

	case c.conf.KeyUndoLastRead:
		c.undoArticle.read = false
		c.prevArticle.read = false
		c.prevArticle = c.undoArticle
		c.ShowArticles(c.activeFeed)
		c.win.articles.Select(1, 3)

	case "h":
		break

	case "l":
		break

	case "Left":
		break

	case "Right":
		break

	default:
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

		// Fallback if no matches
		return e
	}

	return nil
}
