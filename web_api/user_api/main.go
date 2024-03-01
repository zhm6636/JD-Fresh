package main

import (
	"user_api/global"
	_ "user_api/global"
	_ "user_api/router"
)

func main() {
	global.InitWebServer(global.Router)
}
