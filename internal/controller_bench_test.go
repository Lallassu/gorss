package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/k3a/html2text"
)

func setupTestController(b *testing.B, articleCount int) (*Controller, func()) {
	b.Helper()
	tempDir, err := os.MkdirTemp("", "gorss-bench-*")
	if err != nil {
		b.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "bench.db")

	ctrl := &Controller{
		conf: Config{
			Highlights:                    []string{"postgres", "linux", "performance"},
			FeedWindowSizeRatio:           2,
			ArticleWindowSizeRatio:        2,
			PreviewWindowSizeRatio:        1,
			ArticlePreviewWindowSizeRatio: 5,
			FeedNameMaxWidth:              20,
			Feeds: []Feed{
				{URL: "https://example.com/1", Name: "Feed_1"},
				{URL: "https://example.com/2", Name: "Feed_2"},
				{URL: "https://example.com/3", Name: "Feed_3"},
				{URL: "https://example.com/4", Name: "Feed_4"},
				{URL: "https://example.com/5", Name: "Feed_5"},
			},
		},
		theme: Theme{
			FeedBorder:         "#4b7d81",
			ArticleBorder:      "#4b7d81",
			PreviewBorder:      "#4b7d81",
			TableHead:          "#b2b37d",
			Title:              "#fcedd5",
			StatusBackground:   "#4b7d81",
			StatusText:         "#fcedd5",
			StatusKey:          "#f6d270",
			Highlights:         "#c90036",
			Time:               "#f96bad",
			Date:               "#a25478",
			PreviewText:        "#FFFFFF",
			PreviewLink:        "#39537e",
			TotalColumn:        "#FFFFFF",
			UnreadColumn:       "#FFFFFF",
			FeedNames:          []string{"#8ed2c8", "#46aa9f", "#2e6294", "#3b9293", "#a25478"},
			LinkMarker:         "[L]",
			UnreadMarker:       "*",
			FeedIcon:           "[F]",
			ArticleIcon:        "[A]",
			PreviewIcon:        "[P]",
		},
		articles: make([]Article, 0, articleCount),
	}

	ctrl.rss = &RSS{c: ctrl}

	ctrl.db = &DB{}
	if err := ctrl.db.Init(ctrl, dbPath); err != nil {
		b.Fatalf("failed to init db: %v", err)
	}

	ctrl.win = &Window{}
	ctrl.win.Init(func(e *tcell.EventKey) *tcell.EventKey { return e }, ctrl)
	ctrl.win.app.SetFocus(ctrl.win.articles)

	rawHTML := `<div><h1>High Performance Database Internals</h1><p>PostgreSQL WAL storage engine mechanics and B+Tree indexing optimization under heavy write amplification.</p><pre><code>EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM orders WHERE user_id = 42;</code></pre><p>More technical breakdown and benchmark metrics comparing eBPF tracepoints.</p></div>`

	for i := 1; i <= articleCount; i++ {
		art := Article{
			id:          i,
			c:           ctrl,
			feed:        fmt.Sprintf("Feed_%d", (i%5)+1),
			feedDisplay: fmt.Sprintf("Feed %d", (i%5)+1),
			title:       fmt.Sprintf("Technical Article #%d: Linux Kernel eBPF Profiling & Storage", i),
			content:     rawHTML,
			link:        fmt.Sprintf("https://example.com/article/%d", i),
			published:   time.Now().Add(-time.Duration(i) * time.Hour),
			read:        false,
			highlight:   i%3 == 0,
		}
		ctrl.articles = append(ctrl.articles, art)
		ctrl.db.Save(art)
	}

	ctrl.activeFeed = "allarticles"
	ctrl.ShowArticles("allarticles")

	cleanup := func() {
		if ctrl.db != nil {
			ctrl.db.Close()
		}
		os.RemoveAll(tempDir)
	}
	return ctrl, cleanup
}

func BenchmarkSelectArticle_100Articles(b *testing.B) {
	ctrl, cleanup := setupTestController(b, 100)
	defer cleanup()

	idx := 0
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		row := (idx % 100) + 1
		ctrl.win.articles.Select(row, 2)
		ctrl.SelectArticle(row, 2)
		idx++
	}
}

func BenchmarkSelectArticle_500Articles(b *testing.B) {
	ctrl, cleanup := setupTestController(b, 500)
	defer cleanup()

	idx := 0
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		row := (idx % 500) + 1
		ctrl.win.articles.Select(row, 2)
		ctrl.SelectArticle(row, 2)
		idx++
	}
}

func BenchmarkSelectArticle_1000Articles(b *testing.B) {
	ctrl, cleanup := setupTestController(b, 1000)
	defer cleanup()

	idx := 0
	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		row := (idx % 1000) + 1
		ctrl.win.articles.Select(row, 2)
		ctrl.SelectArticle(row, 2)
		idx++
	}
}

func BenchmarkHTML2TextParsing(b *testing.B) {
	rawHTML := `<div><h1>High Performance Database Internals</h1><p>PostgreSQL WAL storage engine mechanics and B+Tree indexing optimization under heavy write amplification.</p><pre><code>EXPLAIN (ANALYZE, BUFFERS) SELECT * FROM orders WHERE user_id = 42;</code></pre><p>More technical breakdown and benchmark metrics comparing eBPF tracepoints and memory allocation profiles across worker threads.</p></div>`

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_ = html2text.HTML2Text(rawHTML)
	}
}
