package main

import (
	"goods_api/global"
	_ "goods_api/global"
	_ "goods_api/router"
)

func main() {
	global.InitWebServer(global.Router)
}
