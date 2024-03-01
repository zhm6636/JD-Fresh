package main

import (
	"inventory_api/global"
	_ "inventory_api/global"
	_ "inventory_api/router"
)

func main() {
	global.InitWebServer(global.Router)
}
