package main

import (
	"flag"
	"io/ioutil"
	"log"
	"strconv"

	"gopkg.in/yaml.v2"
)

type Configuration struct {
	Hostname     string   `yaml:"hostname"`
	Port         int      `yaml:"port"`
	AllowedHosts []string `yaml:"allowed-hosts"`
	Host         string
}

var (
	flagConfigurationFilename string
	flagHostname              string
	flagPort                  int
)

func (c *Configuration) deriveValues() {
	c.Host = c.Hostname + ":" + strconv.Itoa(c.Port)
}

func processConfiguration() *Configuration {
	// Parse CLI flags
	flag.StringVar(&flagConfigurationFilename, "config", "configuration.yaml", "Configuration file to load")
	flag.StringVar(&flagHostname, "hostname", "", "Hostname to listen on")
	flag.IntVar(&flagPort, "port", 0, "Port to listen on")
	flag.Parse()

	// Parse YAML configuration
	configurationFileContents, err := ioutil.ReadFile(flagConfigurationFilename)

	if err != nil {
		log.Fatalln("Error reading configuration YAML:", err)
	}

	var c Configuration

	err = yaml.Unmarshal(configurationFileContents, &c)

	if err != nil {
		log.Fatalln("Error parsing configuration YAML", err)
	}

	// Override YAML configuration with flags
	if len(flagHostname) > 0 {
		c.Hostname = flagHostname
	}

	if flagPort > 0 {
		c.Port = flagPort
	}

	// Derived configurations
	c.deriveValues()

	// Return
	return &c
}
