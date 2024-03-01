package main

import (
	"order_api/global"
	_ "order_api/global"
	_ "order_api/router"
)

func main() {
	global.InitWebServer(global.Router)
}
