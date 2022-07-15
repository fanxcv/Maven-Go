package main

import (
	"Maven-Go/src/util"
	"encoding/json"
	"fmt"
)

var log = util.Log

func main() {
	config := util.LoadConfig()

	if bytes, err := json.Marshal(config); err == nil {
		log.Debugf("启动参数: %s\n", bytes)
	} else {
		log.Panic(err)
	}

	if err := util.Engine.Run(fmt.Sprintf("%s:%s", config.Listen, config.Port)); err != nil {
		log.Errorln("服务启动失败")
	}
}
