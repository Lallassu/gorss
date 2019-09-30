package internal

import (
	"time"
)

// Article holds the content of a feed item
type Article struct {
	c         *Controller
	id        int
	feed      string
	title     string
	content   string
	link      string
	read      bool
	deleted   bool
	highlight bool
	published time.Time
}
