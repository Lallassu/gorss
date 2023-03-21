package internal

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"
)

// Config load the configuration from JSON file
type Config struct {
	Highlights                    []string      `json:"highlights"`
	InputFeeds                    []interface{} `json:"feeds"`
	Feeds                         []Feed        `json:"-"`
	OPMLFile                      string        `json:"opmlFile"`
	FeedWindowSizeRatio           int           `json:"feedWindowSizeRatio"`
	ArticleWindowSizeRatio        int           `json:"articleWindowSizeRatio"`
	PreviewWindowSizeRatio        int           `json:"previewWindowSizeRatio"`
	ArticlePreviewWindowSizeRatio int           `json:"articlePreviewWindowSizeRatio"`
	SecondsBetweenUpdates         int           `json:"secondsBetweenUpdates"`
	SkipArticlesOlderThanDays     int           `json:"skipArticlesOlderThanDays"`
	DaysToKeepDeletedArticlesInDB int           `json:"daysToKeepDeletedArticlesInDB"`
	DaysToKeepReadArticlesInDB    int           `json:"daysToKeepReadArticlesInDB"`
	SkipPreviewInTab              bool          `json:"skipPreviewInTab"`
	KeyOpenLink                   string        `json:"keyOpenLink"`
	KeyMarkLink                   string        `json:"keyMarkLink"`
	KeyOpenMarked                 string        `json:"keyOpenMarked"`
	KeyDeleteArticle              string        `json:"keyDeleteArticle"`
	KeyMoveDown                   string        `json:"keyMoveDown"`
	KeyMoveUp                     string        `json:"keyMoveUp"`
	KeyFeedDown                   string        `json:"keyFeedDown"`
	KeyFeedUp                     string        `json:"keyFeedUp"`
	KeySortByDate                 string        `json:"keySortByDate"`
	KeySortByTitle                string        `json:"keySortByTitle"`
	KeySortByUnread               string        `json:"keySortByUnread"`
	KeySortByFeed                 string        `json:"keySortByFeed"`
	KeyUpdateFeeds                string        `json:"keyUpdateFeeds"`
	KeyMarkAllRead                string        `json:"keyMarkAllRead"`
	KeyMarkAllReadFeed            string        `json:"keyMarkAllReadFeed"`
	KeyMarkAllUnread              string        `json:"keyMarkAllUnread"`
	KeyMarkAllUnreadFeed          string        `json:"keyMarkAllUnreadFeed"`
	KeyTogglePreview              string        `json:"keyTogglePreview"`
	KeySelectFeedWindow           string        `json:"keySelectFeedWindow"`
	KeySelectArticleWindow        string        `json:"keySelectArticleWindow"`
	KeySelectPreviewWindow        string        `json:"keySelectPreviewWindow"`
	KeyToggleHelp                 string        `json:"keyToggleHelp"`
	KeySwitchWindows              string        `json:"keySwitchWindows"`
	KeyQuit                       string        `json:"keyQuit"`
	KeyUndoLastRead               string        `json:"keyUndoLastRead"`
	KeySearchPromt                string        `json:"keySearchPromt"`
	// WebBrowser overrides the default program used to open links. Default one depends on the OS:
	// * `xdg-open` for Linux
	// * `url.dll,FileProtocolHandler` for Windows
	// * `open` for Darwin
	WebBrowser     string    `json:"webBrowser"`
	CustomCommands []Command `json:"customCommands"`
	Notifications  bool      `json:"notifications"`
}

// Feed -
type Feed struct {
	URL  string
	Name string
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

	// Convert Feeds to []Feed{}
	conf.Feeds = make([]Feed, len(conf.InputFeeds))
	for idx := range conf.InputFeeds {
		switch v := conf.InputFeeds[idx].(type) {
		case string:
			// Old style
			conf.Feeds[idx] = Feed{URL: v}
		case map[string]interface{}:
			// New style
			url := v["url"].(string)
			name := ""
			if _, ok := v["name"]; ok {
				name = v["name"].(string)
			}
			conf.Feeds[idx] = Feed{URL: url, Name: name}
		default:
			log.Fatalf("unable to convert %v to a feed", v)
		}
	}

	// Validate that no keys are the same
	keys := make(map[string]struct{})
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
