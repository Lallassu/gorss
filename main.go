package main

import (
	"flag"
)

func main() {
	configFile := flag.String("config", "gorss.conf", "Configuration file")
	themeFile := flag.String("theme", "default.theme", "Theme file")

	flag.Parse()

	co := &Controller{}
	co.Init(*configFile, *themeFile)

}
