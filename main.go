package main

import (
	"github.com/koloo91/performance_monitor/models"
	"encoding/json"
	"os"
	"github.com/koloo91/performance_monitor/services"
	"sync"
)

const (
	configurationFile = "/Users/patrickkolodziej/dev/projects/go/src/github.com/koloo91/performance_monitor/conf/conf.json"
)

var waitGroup sync.WaitGroup

// CPU cat /proc/stat
func main() {
	configuration := loadConfiguration()

	for _, sshConfiguration := range configuration.SshConfigurations {
		go services.Init(&sshConfiguration)
	}

	waitGroup.Add(1)
	waitGroup.Wait()
}

func loadConfiguration() *models.Configuration {
	file, err := os.Open(configurationFile)
	if err != nil {
		panic(err)
	}

	var configuration models.Configuration
	json.NewDecoder(file).Decode(&configuration)

	return &configuration
}
