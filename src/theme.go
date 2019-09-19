package main

import (
	"encoding/json"
	"log"
	"os"
)

// Theme holds all colors for the theme
type Theme struct {
	FeedNames          []string `json:"feedNames"`
	Date               string   `json:"date"`
	Time               string   `json:"time"`
	ArticleBorder      string   `json:"articleBorder"`
	PreviewBorder      string   `json:"previewBorder"`
	FeedBorder         string   `json:"feedBorder"`
	ArticleBorderTitle string   `json:"articleBorderTitle"`
	FeedBorderTitle    string   `json:"feedBorderTitle"`
	PreviewBorderTitle string   `json:"previewBorderTitle"`
	Highlights         string   `json:"highlights"`
	TableHead          string   `json:"tableHead"`
	Title              string   `json:"title"`
	UnreadFeedName     string   `json:"unreadFeedName"`
	TotalColumn        string   `json:"totalColumn"`
	UnreadColumn       string   `json:"unreadColumn"`
	PreviewText        string   `json:"previewText"`
	PreviewLink        string   `json:"previewLink"`
	UnreadMarker       string   `json:"unreadMarker"`
	LinkMarker         string   `json:"linkMarker"`
	FeedIcon           string   `json:"feedIcon"`
	ArticleIcon        string   `json:"articleIcon"`
	PreviewIcon        string   `json:"previewIcon"`
	StatusBackground   string   `json:"statusBackground"`
	StatusText         string   `json:"statusText"`
	StatusKey          string   `json:"statusKey"`
	StatusBrackets     string   `json:"statusBrackets"`
}

// LoadTheme loads a theme file and parses it.
func LoadTheme(file string) Theme {
	var theme Theme
	themeFile, err := os.Open(file)
	defer themeFile.Close()

	if err != nil {
		log.Fatal("Failed to parse theme file:", err)
	}

	jsonParser := json.NewDecoder(themeFile)
	err = jsonParser.Decode(&theme)
	if err != nil {
		log.Fatal("Failed to parse theme file:", err)
	}

	return theme
}
