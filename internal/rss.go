package internal

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gilliek/go-opml/opml"
	"github.com/mmcdole/gofeed"
)

// RSS structure for handle parsing of RSS/Atom feeds
type RSS struct {
	feeds []struct {
		displayName string
		feed        *gofeed.Feed
	}
	c *Controller
}

// Init reads an feed related configuration
func (r *RSS) Init(c *Controller) {
	r.c = c

	// Check if we have any OMPL file to load
	if r.c.conf.OPMLFile != "" {
		doc, err := opml.NewOPMLFromFile(r.c.conf.OPMLFile)
		if err != nil {
			log.Printf("Failed to load OPML file, %v", err)
			return
		}

		// Add URLs to the list of feeds
		for _, b := range doc.Body.Outlines {
			if b.Outlines != nil {
				for _, o := range b.Outlines {
					url := r.GetURLFromOPML(o)
					if url != "" {
						r.c.conf.Feeds = append(r.c.conf.Feeds, Feed{URL: url})
					}
				}
			} else {
				url := r.GetURLFromOPML(b)
				if url != "" {
					r.c.conf.Feeds = append(r.c.conf.Feeds, Feed{URL: url})
				}
			}
		}
	}
}

// GetURLFromOPML retrieves any URL from the OPML object
func (r *RSS) GetURLFromOPML(b opml.Outline) string {
	str := ""
	if b.XMLURL != "" {
		str = b.XMLURL
	} else if b.HTMLURL != "" {
		str = b.HTMLURL
	} else if b.URL != "" {
		str = b.URL
	}
	return str
}

// replaceHTMLEntities replaces common HTML entities with their numeric character references
// to prevent XML parsing errors
func replaceHTMLEntities(content string) string {
	replacements := map[string]string{
		"&nbsp;":   "&#160;",
		"&copy;":   "&#169;",
		"&reg;":    "&#174;",
		"&trade;":  "&#8482;",
		"&mdash;":  "&#8212;",
		"&ndash;":  "&#8211;",
		"&hellip;": "&#8230;",
		"&ldquo;":  "&#8220;",
		"&rdquo;":  "&#8221;",
		"&lsquo;":  "&#8216;",
		"&rsquo;":  "&#8217;",
		"&middot;": "&#183;",
		"&bull;":   "&#8226;",
		"&prime;":  "&#8242;",
		"&Prime;":  "&#8243;",
		"&sect;":   "&#167;",
		"&para;":   "&#182;",
		"&times;":  "&#215;",
		"&divide;": "&#247;",
		"&deg;":    "&#176;",
		"&plusmn;": "&#177;",
		"&sup2;":   "&#178;",
		"&sup3;":   "&#179;",
		"&frac14;": "&#188;",
		"&frac12;": "&#189;",
		"&frac34;": "&#190;",
		"&euro;":   "&#8364;",
		"&pound;":  "&#163;",
		"&yen;":    "&#165;",
		"&cent;":   "&#162;",
		"&iexcl;":  "&#161;",
		"&iquest;": "&#191;",
		"&laquo;":  "&#171;",
		"&raquo;":  "&#187;",
	}

	for entity, replacement := range replacements {
		content = strings.ReplaceAll(content, entity, replacement)
	}

	return content
}

// Update fetches all articles for all feeds
func (r *RSS) Update() {
	fp := gofeed.NewParser()
	r.feeds = []struct {
		displayName string
		feed        *gofeed.Feed
	}{}

	var mu sync.Mutex

	var wg sync.WaitGroup

	for _, f := range r.c.conf.Feeds {
		wg.Add(1)
		go func(f Feed) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("panic while fetching url: %s, panic: %v", f.URL, r)
				}
				wg.Done()
			}()
			feed, err := r.FetchURL(fp, f.URL)
			if err != nil {
				log.Printf("error fetching url: %s, err: %v", f.URL, err)
			} else {
				mu.Lock()
				r.feeds = append(r.feeds, struct {
					displayName string
					feed        *gofeed.Feed
				}{
					f.Name,
					feed,
				})
				mu.Unlock()
			}
		}(f)
	}
	wg.Wait()
}

// FetchURL fetches the feed URL and also fakes the user-agent to be able
// to retrieve data from sites like reddit.
func (r *RSS) FetchURL(fp *gofeed.Parser, url string) (feed *gofeed.Feed, err error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/74.0.3729.169 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp != nil {
		defer func() {
			ce := resp.Body.Close()
			if ce != nil {
				err = ce
			}
		}()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to get url %v, %v", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Clean up from common HTML entities (hopefully solving parse issues)
	cleanedBody := replaceHTMLEntities(string(body))

	return fp.ParseString(cleanedBody)
}
