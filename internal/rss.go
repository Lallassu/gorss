package internal

import (
	"fmt"
	"log"
	"net/http"
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
			wg.Done()
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

	return fp.Parse(resp.Body)
}
