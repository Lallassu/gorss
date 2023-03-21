package internal

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"jaytaylor.com/html2text"
)

// Window holds all information regarding the Window layout and functionality
type Window struct {
	c           *Controller
	feeds       *tview.Table
	articles    *tview.Table
	status      *tview.Table
	help        *tview.Table
	preview     *tview.TextView
	app         *tview.Application
	theme       *Theme
	flexMiddle  *tview.Flex
	flexStatus  *tview.Flex
	flexGlobal  *tview.Flex
	flexFeeds   *tview.Flex
	layout      *tview.Flex
	showPreview bool
	showHelp    bool
	nArticles   int
	nFeeds      int
	askQuit     bool
	currSearch  string
}

const (
	// KeyCell -
	KeyCell = iota
	// ActionCell -
	ActionCell
)

// Init sets up all information regarding widgets
func (w *Window) Init(inputFunc func(*tcell.EventKey) *tcell.EventKey, c *Controller) {
	w.c = c

	w.showPreview = true
	w.showHelp = false

	// Feeds window
	w.feeds = tview.NewTable()
	w.feeds.SetBorder(true)
	w.feeds.SetBorderPadding(1, 1, 1, 1)
	w.feeds.SetBorderColor(tcell.GetColor(w.c.theme.FeedBorder))
	w.feeds.SetTitle(fmt.Sprintf("%s Feeds", w.c.theme.FeedIcon)).SetTitleColor(tcell.GetColor(w.c.theme.FeedBorderTitle))

	// Articles window
	w.articles = tview.NewTable()
	w.articles.SetTitleAlign(tview.AlignLeft)
	w.articles.SetBorder(true)
	w.articles.SetBorderPadding(1, 1, 1, 1)
	w.articles.SetBorderColor(tcell.GetColor(w.c.theme.ArticleBorder))
	w.articles.SetTitle(fmt.Sprintf("%s Articles", w.c.theme.ArticleIcon)).SetTitleColor(tcell.GetColor(w.c.theme.ArticleBorderTitle))

	// Help window
	w.help = tview.NewTable()
	w.help.SetTitleAlign(tview.AlignLeft)
	w.help.SetBorder(true)
	w.help.SetBorderPadding(1, 1, 1, 1)
	w.help.SetBorderColor(tcell.GetColor(w.c.theme.ArticleBorder))
	w.help.SetTitle("💡 Help").SetTitleColor(tcell.GetColor(w.c.theme.ArticleBorderTitle))

	ts := tview.NewTableCell("Key")
	ts.SetAlign(tview.AlignLeft)
	ts.Attributes |= tcell.AttrBold
	ts.SetSelectable(false)
	w.help.SetCell(0, KeyCell, ts)

	ts = tview.NewTableCell("Action")
	ts.SetAlign(tview.AlignLeft)
	ts.Attributes |= tcell.AttrBold
	ts.SetSelectable(false)
	w.help.SetCell(0, ActionCell, ts)

	i := 1
	configKeys := c.GetConfigKeys()
	var keys []string
	for k := range configKeys {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Populate help window
	for _, k := range keys {
		i++
		ts = tview.NewTableCell(fmt.Sprintf("%s", configKeys[k]))
		ts.SetAlign(tview.AlignLeft)
		ts.Attributes |= tcell.AttrBold
		ts.SetSelectable(false)
		ts.SetTextColor(tcell.GetColor(w.c.theme.StatusKey))
		w.help.SetCell(i, KeyCell, ts)

		ts = tview.NewTableCell(fmt.Sprintf("%s", k))
		ts.SetAlign(tview.AlignLeft)
		ts.SetTextColor(tcell.GetColor(w.c.theme.StatusText))
		ts.Attributes |= tcell.AttrBold
		ts.SetSelectable(false)
		w.help.SetCell(i, ActionCell, ts)
	}

	// Preview window
	w.preview = tview.NewTextView()
	w.preview.SetBorder(true)
	w.preview.SetBorderPadding(1, 1, 1, 1)
	w.preview.SetTitleAlign(tview.AlignLeft)
	w.preview.SetBorderColor(tcell.GetColor(w.c.theme.PreviewBorder))
	w.preview.SetScrollable(true)
	w.preview.SetWordWrap(true)
	w.preview.SetDynamicColors(true)
	w.preview.SetTitle(fmt.Sprintf("%s Preview", w.c.theme.PreviewIcon)).SetTitleColor(tcell.GetColor(w.c.theme.PreviewBorderTitle))

	w.status = tview.NewTable()
	w.status.SetBackgroundColor(tcell.GetColor(w.c.theme.StatusBackground))
	w.status.SetFixed(1, 6)

	for i := 0; i < 7; i++ {
		ts = tview.NewTableCell("")
		ts.SetAlign(tview.AlignLeft)
		ts.Attributes |= tcell.AttrBold
		ts.SetSelectable(false)
		w.status.SetCell(0, i, ts)
	}

	w.app = tview.NewApplication()
	w.app.SetInputCapture(inputFunc)

	w.UpdateStatusTicker()
	w.SetupWindow()
}

// RegisterSelectedFeedFunc registers a hook function for selecting feed
func (w *Window) RegisterSelectedFeedFunc(f func(r, c int)) {
	w.feeds.SetSelectionChangedFunc(f)
}

// RegisterSelectedFunc registers a hook function for selected article
func (w *Window) RegisterSelectedFunc(f func(r, c int)) {
	w.articles.SetSelectedFunc(f)
}

// RegisterSelectionChangedFunc registers a hook for change events for articles
func (w *Window) RegisterSelectionChangedFunc(f func(r, c int)) {
	w.articles.SetSelectionChangedFunc(f)
}

// SetupWindow initiates the window layout
func (w *Window) SetupWindow() {
	w.flexMiddle = tview.NewFlex().SetDirection(tview.FlexRow)
	w.flexMiddle.AddItem(w.articles, 0, w.c.conf.ArticleWindowSizeRatio, false)
	w.flexMiddle.AddItem(w.preview, 0, w.c.conf.PreviewWindowSizeRatio, false)

	w.flexFeeds = tview.NewFlex().SetDirection(tview.FlexRow)
	w.flexFeeds.AddItem(w.feeds, 0, 1, false)

	w.flexGlobal = tview.NewFlex().SetDirection(tview.FlexColumn)
	w.flexGlobal.AddItem(w.flexFeeds, 0, w.c.conf.FeedWindowSizeRatio, false)
	w.flexGlobal.AddItem(w.flexMiddle, 0, w.c.conf.ArticlePreviewWindowSizeRatio, false)

	w.flexStatus = tview.NewFlex().SetDirection(tview.FlexRow)
	w.flexStatus.AddItem(w.flexGlobal, 0, 20, false)
	w.flexStatus.AddItem(w.status, 1, 1, false)

	w.layout = tview.NewFlex()
	w.layout.AddItem(w.flexStatus, 0, 1, false)
}

// Start initiates the application for the window system and sets its root
func (w *Window) Start() {
	if err := w.app.SetRoot(w.layout, true).SetFocus(w.articles).Run(); err != nil {
		panic(err)
	}
}

// ToggleHelp shows/hides the keyboard shortchuts
func (w *Window) ToggleHelp() {
	if !w.showHelp {
		w.flexMiddle = w.flexMiddle.RemoveItem(w.preview)
		w.flexMiddle = w.flexMiddle.RemoveItem(w.articles)
		w.flexMiddle = w.flexMiddle.AddItem(w.help, 0, 1, false)
		w.showHelp = true
	} else {
		w.flexMiddle = w.flexMiddle.AddItem(w.articles, 0, 5, false)
		w.flexMiddle = w.flexMiddle.AddItem(w.preview, 0, 1, false)
		w.flexMiddle = w.flexMiddle.RemoveItem(w.help)
		w.showHelp = false
	}
}

// TogglePreview shows/hides the preview window for an article
func (w *Window) TogglePreview() {
	if !w.showPreview {
		w.flexMiddle = w.flexMiddle.AddItem(w.preview, 0, w.c.conf.PreviewWindowSizeRatio, false)
		w.showPreview = true
	} else {
		w.flexMiddle = w.flexMiddle.RemoveItem(w.preview)
		w.showPreview = false
	}
}

// UpdateStatusTicker calls StatusUpdate periodically
func (w *Window) UpdateStatusTicker() {
	w.StatusUpdate()
	ticker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.StatusUpdate()
			}
		}
	}()
}

// Search asks the user to input search query
func (w *Window) Search() {
	w.askQuit = true
	w.flexStatus.RemoveItem(w.status)
	w.currSearch = ""

	inputField := tview.NewInputField().
		SetLabel("find: ").
		SetFieldWidth(20).
		SetFieldBackgroundColor(tcell.ColorBlack)

	capt := func(e *tcell.EventKey) *tcell.EventKey {
		keyName := string(e.Name())
		if strings.Contains(keyName, "Rune") {
			keyName = string(e.Rune())
		}

		if strings.EqualFold(keyName, "esc") {
			w.flexStatus.RemoveItem(inputField)
			w.flexStatus.AddItem(w.status, 1, 1, false)
			w.app.SetInputCapture(w.c.Input)
			w.app.SetFocus(w.articles)
		}

		if strings.EqualFold(keyName, "enter") {
			w.flexStatus.RemoveItem(inputField)
			w.flexStatus.AddItem(w.status, 1, 1, false)
			w.app.SetInputCapture(w.c.Input)
			w.app.SetFocus(w.articles)

			w.feeds.Select(2, 0)
			w.articles.Select(0, 3)

		} else {
			w.currSearch += keyName
		}

		return e
	}
	w.flexStatus.AddItem(inputField, 1, 0, false)
	w.app.SetFocus(inputField)
	w.app.SetInputCapture(capt)
}

// AskQuit asks the user to quit or not.
func (w *Window) AskQuit() {
	w.askQuit = true
	w.flexStatus.RemoveItem(w.status)

	inputField := tview.NewInputField().
		SetLabel("Quit [Y/n]? ").
		SetFieldWidth(5).
		SetFieldBackgroundColor(tcell.ColorBlack)

	x := func(e *tcell.EventKey) *tcell.EventKey {
		keyName := string(e.Name())
		if strings.Contains(keyName, "Rune") {
			keyName = string(e.Rune())
		}

		if strings.EqualFold(keyName, "y") || strings.EqualFold(keyName, "enter") {
			w.c.quit <- 1
		}

		w.flexStatus.RemoveItem(inputField)
		w.flexStatus.AddItem(w.status, 1, 1, false)
		w.app.SetInputCapture(w.c.Input)
		w.app.SetFocus(w.articles)

		return e
	}
	w.flexStatus.AddItem(inputField, 1, 0, false)
	w.app.SetFocus(inputField)
	w.app.SetInputCapture(x)
}

// StatusUpdate updates the status window with updated information
func (w *Window) StatusUpdate() {
	if w.askQuit {
		return
	}
	// Update time
	c := w.status.GetCell(0, 0)
	c.SetText(
		fmt.Sprintf(
			"[%s][[%s]Time: [%s]%s[%s]]",
			w.c.theme.StatusBrackets,
			w.c.theme.StatusKey,
			w.c.theme.StatusText,
			time.Now().Format("15:04"),
			w.c.theme.StatusBrackets,
		),
	)

	// Last updated
	c = w.status.GetCell(0, 1)
	c.SetText(
		fmt.Sprintf(
			"[%s][[%s]Last Update: [%s]%s[%s]]",
			w.c.theme.StatusBrackets,
			w.c.theme.StatusKey,
			w.c.theme.StatusText,
			w.c.lastUpdate.Format("15:04"),
			w.c.theme.StatusBrackets,
		),
	)
	c = w.status.GetCell(0, 2)

	c.SetText(
		fmt.Sprintf(
			"[%s][[%s]Total Articles: [%s]%d[%s]]",
			w.c.theme.StatusBrackets,
			w.c.theme.StatusKey,
			w.c.theme.StatusText,
			len(w.c.articles),
			w.c.theme.StatusBrackets,
		),
	)

	c = w.status.GetCell(0, 3)
	unread := 0
	feeds := make(map[string]struct{})
	for _, a := range w.c.articles {
		if _, ok := feeds[a.feed]; !ok {
			feeds[a.feed] = struct{}{}
		}
		if !a.read {
			unread++
		}
	}
	c.SetText(
		fmt.Sprintf(
			"[%s][[%s]Total Unread: [%s]%d[%s]]",
			w.c.theme.StatusBrackets,
			w.c.theme.StatusKey,
			w.c.theme.StatusText,
			unread,
			w.c.theme.StatusBrackets,
		),
	)

	c = w.status.GetCell(0, 4)
	c.SetText(
		fmt.Sprintf(
			"[%s][[%s]Feeds: [%s]%d[%s]]",
			w.c.theme.StatusBrackets,
			w.c.theme.StatusKey,
			w.c.theme.StatusText,
			len(feeds),
			w.c.theme.StatusBrackets,
		),
	)

	c = w.status.GetCell(0, 5)
	c.SetText(
		fmt.Sprintf(
			"[%s][[%s]Help: [%s]%s[%s]]",
			w.c.theme.StatusBrackets,
			w.c.theme.StatusKey,
			w.c.theme.StatusText,
			w.c.conf.KeyToggleHelp,
			w.c.theme.StatusBrackets,
		),
	)

	c = w.status.GetCell(0, 6)
	c.SetText(
		fmt.Sprintf(
			"[%s][[%s]Version: [%s]%s[%s]]",
			w.c.theme.StatusBrackets,
			w.c.theme.StatusKey,
			w.c.theme.StatusText,
			Version,
			w.c.theme.StatusBrackets,
		),
	)

	go w.app.Draw()
}

// ClearPreview clears the preview window
func (w *Window) ClearPreview() {
	w.preview.Clear()
}

// ClearArticles resets the articles window
func (w *Window) ClearArticles() {
	w.nArticles = 0
	w.articles.Clear()

	ts := tview.NewTableCell("")
	ts.SetAlign(tview.AlignLeft)
	ts.Attributes |= tcell.AttrBold
	ts.SetSelectable(false)
	w.articles.SetCell(0, 0, ts)

	ts = tview.NewTableCell("Feed")
	ts.SetAlign(tview.AlignLeft)
	ts.Attributes |= tcell.AttrBold
	ts.SetTextColor(tcell.GetColor(w.c.theme.TableHead))
	ts.SetSelectable(false)
	w.articles.SetCell(0, 1, ts)

	ts = tview.NewTableCell("Title")
	ts.Attributes |= tcell.AttrBold
	ts.SetTextColor(tcell.GetColor(w.c.theme.TableHead))
	ts.SetSelectable(false)
	w.articles.SetCell(0, 2, ts)

	ts = tview.NewTableCell("Published")
	ts.Attributes |= tcell.AttrBold
	ts.SetTextColor(tcell.GetColor(w.c.theme.TableHead))
	ts.SetSelectable(false)
	w.articles.SetCell(0, 3, ts)

	w.articles.SetSelectable(true, false)
}

// ClearFeeds resets the feed window
func (w *Window) ClearFeeds() {
	w.feeds.Clear()
	w.feeds.SetTitle(fmt.Sprintf("%s Feeds", w.c.theme.FeedIcon)).SetTitleColor(tcell.GetColor(w.c.theme.FeedBorderTitle))
	w.nFeeds = 0

	ts := tview.NewTableCell("Total")
	ts.SetAlign(tview.AlignLeft)
	ts.Attributes |= tcell.AttrBold
	ts.SetSelectable(false)
	ts.SetTextColor(tcell.GetColor(w.c.theme.TableHead))
	w.feeds.SetCell(0, 0, ts)

	ts = tview.NewTableCell("Unread")
	ts.SetAlign(tview.AlignLeft)
	ts.Attributes |= tcell.AttrBold
	ts.SetSelectable(false)
	ts.SetTextColor(tcell.GetColor(w.c.theme.TableHead))
	w.feeds.SetCell(0, 1, ts)

	ts = tview.NewTableCell("Feed")
	ts.SetAlign(tview.AlignLeft)
	ts.Attributes |= tcell.AttrBold
	ts.SetTextColor(tcell.GetColor(w.c.theme.TableHead))
	ts.SetSelectable(false)
	w.feeds.SetCell(0, 2, ts)

	w.feeds.SetSelectable(true, false)
}

// AddToFeeds add a new feed to the feed window
func (w *Window) AddToFeeds(name, displayName string, unread, total int, ref *Article) {
	w.nFeeds++

	color := "white"
	if len(w.c.conf.Feeds) < len(w.c.theme.FeedNames) {
		idx := 0
		for i, f := range w.c.rss.feeds {
			if f.feed.Title == name {
				idx = i
				break
			}
		}
		color = w.c.theme.FeedNames[idx]
	}

	// Display total number of articles
	nc := tview.NewTableCell(fmt.Sprintf("%d", total))
	nc.SetAlign(tview.AlignLeft)
	w.feeds.SetCell(w.nFeeds, 0, nc)
	nc.SetSelectable(true)
	nc.SetTextColor(tcell.GetColor(w.c.theme.TotalColumn))

	// Display number of unread articles
	nc = tview.NewTableCell(fmt.Sprintf("%d", unread))
	nc.SetAlign(tview.AlignLeft)
	w.feeds.SetCell(w.nFeeds, 1, nc)
	nc.SetSelectable(true)
	nc.SetTextColor(tcell.GetColor(w.c.theme.UnreadColumn))

	// Display the name of the feed
	if displayName == "" {
		displayName = name
	}
	nc = tview.NewTableCell(fmt.Sprintf("%s", displayName))
	nc.SetAlign(tview.AlignLeft)
	w.feeds.SetCell(w.nFeeds, 2, nc)
	nc.SetSelectable(true)
	nc.SetTextColor(tcell.GetColor(color))
	nc.SetReference(ref)
}

// ArticlesHasFocus returns true if the aricles window has focus
func (w *Window) ArticlesHasFocus() bool {
	if w.app.GetFocus() == w.articles {
		return true
	}
	return false
}

// MoveDown handles a keypress for moving down in feeds/articles
func (w *Window) MoveDown(focus tview.Primitive) {
	// Article window
	if focus == w.articles {
		count := w.articles.GetRowCount()
		r, _ := w.articles.GetSelection()
		a := w.c.GetArticleForSelection()
		if r == 1 {
			w.c.db.MarkRead(a)
			a.read = true
		}
		if r < count-1 {
			w.articles.Select(r+1, 3)
		}
	} else if focus == w.feeds {
		// Feed window
		count := w.feeds.GetRowCount()
		r, _ := w.feeds.GetSelection()
		if r < count-1 {
			w.feeds.Select(r+1, 0)
		}
		// Set selected article to first article in feed
		w.articles.Select(0, 3)
	} else if focus == w.preview {
		// Preview window
		r, _ := w.preview.GetScrollOffset()
		w.preview.ScrollTo(r+1, 0)
	}
}

// MoveUp handles a keypress for moving up in feeds/articles
func (w *Window) MoveUp(focus tview.Primitive) {
	if focus == w.articles {
		r, _ := w.articles.GetSelection()
		if r > 1 {
			w.articles.Select(r-1, 3)
		}
	} else if focus == w.feeds {
		r, _ := w.feeds.GetSelection()
		if r > 1 {
			w.feeds.Select(r-1, 0)
		}
	} else if focus == w.preview {
		r, _ := w.preview.GetScrollOffset()
		w.preview.ScrollTo(r-1, 0)
	}
}

// SwitchFocus switches the focus between windows in a round-robin manner
func (w *Window) SwitchFocus() {
	p := w.app.GetFocus()
	if p == w.feeds {
		w.app.SetFocus(w.articles)
	} else if p == w.articles {
		if w.c.conf.SkipPreviewInTab {
			w.app.SetFocus(w.feeds)
		} else {
			w.app.SetFocus(w.preview)
		}
	} else if p == w.preview {
		w.app.SetFocus(w.feeds)
	}
}

// SelectFeedWindow selects the feed window (focus)
func (w *Window) SelectFeedWindow() {
	w.app.SetFocus(w.feeds)
}

// SelectArticleWindow selects the article window (focus)
func (w *Window) SelectArticleWindow() {
	w.app.SetFocus(w.articles)
}

// SelectPreviewWindow selects the preview window (focus)
func (w *Window) SelectPreviewWindow() {
	w.app.SetFocus(w.preview)
}

// AddToArticles adds an article to the article window
func (w *Window) AddToArticles(a *Article, markedWeb bool) {
	if a == nil {
		return
	}
	w.nArticles++

	nc := tview.NewTableCell("")
	nc.SetAlign(tview.AlignLeft)
	w.articles.SetCell(w.nArticles, 0, nc)
	nc.SetSelectable(false)

	// Create different color per feed name
	color := "white"
	if len(w.c.conf.Feeds) < len(w.c.theme.FeedNames) {
		idx := 0
		for i, f := range w.c.rss.feeds {
			if f.feed.Title == a.feed {
				idx = i
				break
			}
		}
		color = w.c.theme.FeedNames[idx]
	}
	fc := tview.NewTableCell(fmt.Sprintf("[%s]%s", color, a.feed))
	fc.SetTextColor(tcell.GetColor(color))
	fc.SetAlign(tview.AlignLeft)
	fc.SetMaxWidth(20)
	w.articles.SetCell(w.nArticles, 1, fc)

	tc := tview.NewTableCell("")
	tc.SetTextColor(tcell.GetColor(w.c.theme.Title))
	tc.SetSelectable(true)
	tc.SetMaxWidth(80)
	tc.SetAlign(tview.AlignLeft)
	tc.SetReference(a)
	w.articles.SetCell(w.nArticles, 2, tc)

	if w.c.activeFeed == "result" {
		hTitle := ""
		fields := strings.Fields(a.title)
		for _, f := range fields {
			found := false
			for _, h := range strings.Fields(w.currSearch) {
				if strings.Contains(strings.ToLower(f), strings.ToLower(h)) {
					found = true
				}
			}
			if found {
				hTitle += fmt.Sprintf("[%s]"+f+" [%s]", w.c.theme.Highlights, w.c.theme.Title)
			} else {
				hTitle += f + " "
			}
		}
		tc.SetText(hTitle)
	} else {
		if a.highlight {
			hTitle := ""
			fields := strings.Fields(a.title)
			for _, f := range fields {
				found := false
				for _, h := range w.c.conf.Highlights {
					if strings.Contains(strings.ToLower(f), strings.ToLower(h)) {
						found = true
					}
				}
				if found {
					hTitle += fmt.Sprintf("[%s]"+f+" [%s]", w.c.theme.Highlights, w.c.theme.Title)
				} else {
					hTitle += f + " "
				}
			}
			tc.SetText(hTitle)
		} else {
			tc.SetText(a.title)
		}
	}

	str := time.Since(a.published).Round(time.Minute).String()
	t := GetTime(str)

	dc := tview.NewTableCell(
		fmt.Sprintf(
			"%s ([%s]%s[%s])",
			a.published.Format("2006-01-02 15:04:05"),
			w.c.theme.Time,
			t,
			w.c.theme.Date,
		),
	)
	dc.SetTextColor(tcell.GetColor(w.c.theme.Date))
	dc.SetAlign(tview.AlignLeft)
	w.articles.SetCell(w.nArticles, 3, dc)

	ncText := ""
	if markedWeb {
		ncText += w.c.theme.LinkMarker
	}
	if !a.read {
		ncText += w.c.theme.UnreadMarker
		fc.Attributes |= tcell.AttrBold
		tc.Attributes |= tcell.AttrBold
		dc.Attributes |= tcell.AttrBold
	}
	nc.SetText(ncText)
}

// AddPreview shows an article in the preview window
func (w *Window) AddPreview(a *Article) {
	parsed, err := html2text.FromString(a.content, html2text.Options{PrettyTables: true})
	if err != nil {
		log.Printf("Failed to parse html to text, rendering original.")
		parsed = a.content
	}

	w.preview.Clear()

	text := fmt.Sprintf(
		"[%s][%s][%s] %s [white]([%s]%s[white])\n\n[%s]%s\n\nLink: [%s]%s",
		"white",
		a.feed,
		w.c.theme.Title,
		a.title,
		w.c.theme.Date,
		a.published,
		w.c.theme.PreviewText,
		parsed,
		w.c.theme.PreviewLink,
		a.link,
	)
	w.preview.SetText(text)
	w.preview.ScrollToBeginning()
}

// GetTime returns the timestring formatted as (%h%m < 24 hours < %d)
func GetTime(ts string) string {
	dDrex := regexp.MustCompile(`(\d+)h`)
	dRes := dDrex.FindStringSubmatch(ts)
	if len(dRes) > 0 {
		if i, err := strconv.Atoi(dRes[1]); err == nil {
			if i > 23 {
				days := i / 24
				return strconv.Itoa(days) + "d"
			}
		}
	}

	rex := regexp.MustCompile(`.*m(.*?)`)
	res := rex.FindAllStringSubmatch(ts, -1)
	if len(res) > 0 && res[0] != nil {
		return string(res[0][0])
	}

	return "-"
}
