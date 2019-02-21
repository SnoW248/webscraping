package main

import (
	"webscraping/app/controllers"
	"webscraping/utils"
)

func main() {
	utils.LoggingSettings("webscraping.log")
	// GetPage("https://tabelog.com/rstLst/ramen/")
	// GetInfo("ramen", 60, "json")
	controllers.StartWebServer()
}
