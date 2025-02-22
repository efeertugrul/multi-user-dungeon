package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
)

type Configuration struct {
	Port           uint16 `json:"Port"`
	UserPoolID     string `json:"UserPoolId"`
	ClientSecret   string `json:"UserPoolClientSecret"`
	UserPoolRegion string `json:"UserPoolRegion"`
	ClientID       string `json:"UserPoolClientId"`
	DataFile       string `json:"DataFile"`
}

func main() {
	// Read configuration file
	configFile := flag.String("config", "config.json", "Configuration file")
	flag.Parse()

	config := Configuration{}
	data, err := os.ReadFile(*configFile)
	if err != nil {
		log.Printf("Failed to read configuration file: %v", err)
		os.Exit(1)
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		log.Printf("Failed to parse configuration file: %v", err)
		os.Exit(1)
	}

	//log.Printf("Configuration loaded: %+v", config)

	// Initialize the Database

	server, err := NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start the server
	if err := server.StartSSHServer(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
