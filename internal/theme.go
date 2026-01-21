package internal

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Lallassu/gorss"
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
// First tries to load from file system, then falls back to embedded themes.
func LoadTheme(file string) Theme {
	var theme Theme

	// Try to load from file system first
	themeFile, err := os.Open(file)
	if err == nil {
		defer themeFile.Close()
		jsonParser := json.NewDecoder(themeFile)
		err = jsonParser.Decode(&theme)
		if err != nil {
			log.Fatal("Failed to parse theme file:", err)
		}
		return theme
	}

	// Fall back to embedded themes
	themeName := filepath.Base(file)
	return LoadEmbeddedTheme(themeName)
}

// LoadEmbeddedTheme loads a theme from embedded filesystem
func LoadEmbeddedTheme(themeName string) Theme {
	var theme Theme

	if !strings.HasSuffix(themeName, ".theme") {
		themeName += ".theme"
	}

	data, err := gorss.EmbeddedThemes.ReadFile("themes/" + themeName)
	if err != nil {
		log.Fatalf("Failed to load embedded theme %s: %v", themeName, err)
	}

	err = json.Unmarshal(data, &theme)
	if err != nil {
		log.Fatalf("Failed to parse embedded theme %s: %v", themeName, err)
	}

	return theme
}

// GetAvailableThemes returns a sorted list of available theme names (without .theme extension)
func GetAvailableThemes() []string {
	entries, err := gorss.EmbeddedThemes.ReadDir("themes")
	if err != nil {
		log.Printf("Failed to read embedded themes directory: %v", err)
		return []string{"default"}
	}

	var themes []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".theme") {
			themeName := strings.TrimSuffix(entry.Name(), ".theme")
			themes = append(themes, themeName)
		}
	}

	sort.Strings(themes)
	return themes
}
