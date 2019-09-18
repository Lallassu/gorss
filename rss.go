package main

// https://godoc.org/github.com/mmcdole/gofeed#example-Parser-ParseString

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"log"
	"net/http"
)

type RSS struct {
	feeds []*gofeed.Feed
	c     *Controller
}

func (r *RSS) Init(c *Controller) {
	r.c = c
}

func (r *RSS) Update() {
	fp := gofeed.NewParser()
	r.feeds = []*gofeed.Feed{}
	for _, f := range r.c.conf.Feeds {
		//feed, _ := fp.ParseURL(f)
		feed, err := r.FetchURL(fp, f)
		if err != nil {
			log.Printf("error fetching url: %s, err: %v", f, err)
			continue
		}
		r.feeds = append(r.feeds, feed)
	}
}

// Do a little dance to fake user agent.
func (r *RSS) FetchURL(fp *gofeed.Parser, url string) (feed *gofeed.Feed, err error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// We can't fetch w/o faking user-agent for sites such as Reddit :/
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
		return nil, fmt.Errorf("Failed to get url %v, %v", resp.StatusCode, resp.Status)
	}

	return fp.Parse(resp.Body)
}
