package config

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/viper"
)

const (
	redPathsConfigName   = "redpaths"
	redPathsConfigPrefix = ".redpaths"
)

func initConfig() {
	viper.AddConfigPath(moduleConfigPath)
	viper.SetConfigName(redPathsConfigName)
	viper.SetConfigType("yaml")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("error while parsing redpaths configuration: %w", err))
	}
}

func RestPort() string {
	initConfig()
	if os.Getenv("REST_PORT") != "" {
		log.Println("Using REST port: " + os.Getenv("REST_PORT"))
		return os.Getenv("REST_PORT")
	}
	log.Printf("-----------" + viper.GetString(redPathsConfigPrefix+".rest_port"))
	return viper.GetString(redPathsConfigPrefix + ".rest_port")
}

func SSEPort() string {
	initConfig()
	if os.Getenv("SSE_PORT") != "" {
		return os.Getenv("SSE_PORT")
	}
	return viper.GetString(redPathsConfigPrefix + ".sse_port")
}
