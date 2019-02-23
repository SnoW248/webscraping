package main

import (
	"webscraping/app/controllers"
	"webscraping/app/models"
	"webscraping/utils"
)

func main() {
	utils.LoggingSettings("webscraping.log")
	// GetPage("https://tabelog.com/rstLst/ramen/")
	models.GetInfo("ramen", 60, "json")
	controllers.StartWebServer()
}
