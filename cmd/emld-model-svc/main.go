/*
 * File: main.go
 * Project: emld-model-svc
 * File Created: Monday, 22nd March 2021 7:52:01 pm
 * Author: Anonymous (anonymous@gmail.com)
 * -----
 * Last Modified: Friday, 2nd February 2024 2:33:11 pm
 * Modified By: Anonymous (anonymous@gmail.com>)
 */
package main

import (
	"fmt"
	"os"

	config "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/common/config"
	svc "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model"
	modelConfig "gitlab.com/krydus/emeraldai/go-emerald-app/pkg/services/model/config"
)

func main() {

	path, exists := os.LookupEnv("CONFIG_PATH")
	if !exists {
		panic(fmt.Errorf("required environmental variable `CONFIG_PATH` unset"))
	}

	conf, err := config.NewConfiguration[modelConfig.Configuration](path, true)
	checkErr(err)

	checkErr(svc.Start(conf))
}

func checkErr(err error) {
	if err != nil {
		panic(err.Error())
	}
}
