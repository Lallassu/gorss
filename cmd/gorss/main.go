package main

import (
	"flag"
	"fmt"
	"github.com/OpenPeeDeeP/xdg"
	"log"
	"os"

	"github.com/Lallassu/gorss/internal"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	defaultConfig := "gorss.conf"
	defaultTheme := "themes/default.theme"
	defaultDB := "gorss.db"
	configFile := flag.String("config", defaultConfig, "Configuration file")
	themeFile := flag.String("theme", defaultTheme, "Theme file")
	dbFile := flag.String("db", defaultDB, "Database file")
	versionFlag := flag.Bool("version", false, "Show version")

	flag.Parse()

	cfg := *configFile
	theme := *themeFile
	db := *dbFile

	if *versionFlag {
		fmt.Printf("GORSS version: %s", internal.Version)
		os.Exit(0)
	}

	conf := xdg.New("", "gorss")

	dataHome := conf.DataHome()
	configHome := conf.ConfigHome()

	// Create dirs
	for _, path := range []string{dataHome, configHome} {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.Mkdir(path, 0700); err != nil {
				log.Printf("Failed to create dir: %s\n", path)
			}
		}
	}

	if cfg == defaultConfig {
		// Try to get using XDG
		s := conf.QueryConfig(defaultConfig)
		if s != "" {
			cfg = s
		}
	}

	if theme == defaultTheme {
		// Try to get using XDG
		s := conf.QueryConfig(defaultTheme)
		if s != "" {
			theme = s
		}
	}

	if db == defaultDB {
		// Try to get using XDG
		s := conf.QueryData(defaultDB)
		if s != "" {
			db = s
		} else {
			db = fmt.Sprintf("%s/%s", conf.DataHome(), defaultDB)
		}
	}

	log.Printf("Using config: %s\n", cfg)
	log.Printf("Using theme: %s\n", theme)
	log.Printf("Using DB: %s\n", db)

	co := &internal.Controller{}

	co.Init(cfg, theme, db)

}
