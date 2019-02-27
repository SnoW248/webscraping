package main

import (
	"webscraping/app/controllers"
	"webscraping/utils"
)

func main() {
	utils.LoggingSettings("webscraping.log")
	controllers.StartWebServer()
}
