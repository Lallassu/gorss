package main

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"
)

// Config load the configuration from JSON file
type Config struct {
	Highlights                    []string  `json:"highlights"`
	Feeds                         []string  `json:"feeds"`
	OPMLFile                      string    `json:"opmlFile"`
	FeedWindowSizeRatio           int       `json:"feedWindowSizeRatio"`
	ArticleWindowSizeRatio        int       `json:"articleWindowSizeRatio"`
	PreviewWindowSizeRatio        int       `json:"previewWindowSizeRatio"`
	ArticlePreviewWindowSizeRatio int       `json:"articlePreviewWindowSizeRatio"`
	SecondsBetweenUpdates         int       `json:"secondsBetweenUpdates"`
	SkipArticlesOlderThanDays     int       `json:"skipArticlesOlderThanDays"`
	DaysToKeepDeletedArticlesInDB int       `json:"daysToKeepDeletedArticlesInDB"`
	DaysToKeepReadArticlesInDB    int       `json:"daysToKeepReadArticlesInDB"`
	SkipPreviewInTab              bool      `json:"skipPreviewInTab"`
	KeyOpenLink                   string    `json:"keyOpenLink"`
	KeyMarkLink                   string    `json:"keyMarkLink"`
	KeyOpenMarked                 string    `json:"keyOpenMarked"`
	KeyDeleteArticle              string    `json:"keyDeleteArticle"`
	KeyMoveDown                   string    `json:"keyMoveDown"`
	KeyMoveUp                     string    `json:"keyMoveUp"`
	KeySortByDate                 string    `json:"keySortByDate"`
	KeySortByTitle                string    `json:"keySortByTitle"`
	KeySortByUnread               string    `json:"keySortByUnread"`
	KeySortByFeed                 string    `json:"keySortByFeed"`
	KeyUpdateFeeds                string    `json:"keyUpdateFeeds"`
	KeyMarkAllRead                string    `json:"keyMarkAllRead"`
	KeyMarkAllUnread              string    `json:"keyMarkAllUnread"`
	KeyTogglePreview              string    `json:"keyTogglePreview"`
	KeySelectFeedWindow           string    `json:"keySelectFeedWindow"`
	KeySelectArticleWindow        string    `json:"keySelectArticleWindow"`
	KeySelectPreviewWindow        string    `json:"keySelectPreviewWindow"`
	KeyToggleHelp                 string    `json:"keyToggleHelp"`
	KeySwitchWindows              string    `json:"keySwitchWindows"`
	KeyQuit                       string    `json:"keyQuit"`
	KeyUndoLastRead               string    `json:"keyUndoLastRead"`
	CustomCommands                []Command `json:"customCommands"`
}

// Command is used to parse a custom key->command from configuration file.
type Command struct {
	Key  string
	Cmd  string
	Args string
}

// LoadConfiguration takes a filename (configuration) and loads it.
func LoadConfiguration(file string) Config {
	var conf Config
	configFile, err := os.Open(file)
	defer configFile.Close()

	if err != nil {
		log.Fatal("Failed to open config file:", err)
	}

	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&conf)
	if err != nil {
		log.Fatal("Failed to parse config file:", err)
	}

	// Validate that no keys are the same
	keys := make(map[string]struct{}, 0)
	val := reflect.Indirect(reflect.ValueOf(conf))
	for i := 0; i < val.NumField(); i++ {
		if strings.HasPrefix(val.Type().Field(i).Name, "Key") {
			if _, ok := keys[val.Field(i).String()]; ok {
				log.Fatal("Key defined more than once, key: ", val.Field(i).String())
			} else {
				keys[val.Field(i).String()] = struct{}{}
			}
		}
	}

	// Then check custom commands as well
	for _, cmd := range conf.CustomCommands {
		if _, ok := keys[cmd.Key]; ok {
			log.Fatal("Key defined more than once, key: ", cmd.Key)
		} else {
			keys[cmd.Key] = struct{}{}
		}
	}

	return conf
}
