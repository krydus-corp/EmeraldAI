package main

import (
	"fmt"
	"os"

	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/config"
	svc "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal"
	portalConfig "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/portal/config"
)

func main() {

	path, exists := os.LookupEnv("CONFIG_PATH")
	if !exists {
		panic(fmt.Errorf("required environmental variable `CONFIG_PATH` unset"))
	}

	conf, err := config.NewConfiguration[portalConfig.Configuration](path, true)
	checkErr(err)

	checkErr(svc.Start(conf))
}

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
