package main

import (
	"time"
)

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
