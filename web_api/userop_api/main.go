package main

import (
	"userop_api/global"
	_ "userop_api/global"
	_ "userop_api/router"
)

func main() {
	global.InitWebServer(global.Router)
}
