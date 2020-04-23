package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/OpenPeeDeeP/xdg"

	"github.com/Lallassu/gorss/internal"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	defaultConfig := "gorss.conf"
	defaultTheme := "themes/default.theme"
	defaultDB := "gorss.db"
	defaultLog := "gorss.log"

	configFile := flag.String("config", defaultConfig, "Configuration file")
	themeFile := flag.String("theme", defaultTheme, "Theme file")
	logFile := flag.String("log", defaultLog, "Log file")
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

	if *logFile == defaultLog {
		// Try to get using XDG
		s := conf.QueryData(defaultLog)
		if s != "" {
			*logFile = s
		} else {
			*logFile = fmt.Sprintf("%s/%s", conf.DataHome(), defaultLog)
		}
	}

	log.Printf("Using config: %s\n", cfg)
	log.Printf("Using theme: %s\n", theme)
	log.Printf("Using DB: %s\n", db)
	log.Printf("Using log file: %s\n", *logFile)

	if flog, err := os.Create(*logFile); err != nil {
		log.Printf("Failed to create log file. Will log to stdout.")
	} else {
		log.SetOutput(flog)
	}
	co := &internal.Controller{}

	co.Init(cfg, theme, db)

}
