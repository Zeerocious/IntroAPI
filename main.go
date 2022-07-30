package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
)

var config *Config

const (
	TWITTER_API_BASE = "https://api.twitter.com"
)

type User struct {
	Data struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	} `json:"data"`
}
type Config struct {
	Twitter struct {
		Bearer string `json:"Bearer_Token"`
	} `json:"twitter"`
}

func main() {
	configPath, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	config, err = NewConfig(configPath)
	if err != nil {
		log.Fatal(err)
	}

	/*client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+config.Twitter.Bearer).
		Get(TWITTER_API_BASE + "/2/users/by/username/Overpowered")
	if err != nil {
		fmt.Println(err)
	}
	var OP User
	json.Unmarshal(resp.Body(), &OP)
	fmt.Println(OP.Data.ID)*/

}
func ParseFlags() (string, error) {
	var configPath string
	flag.StringVar(&configPath, "config", "./config.json", "path to config file")
	flag.Parse()

	if err := ValidateConfigPath(configPath); err != nil {
		return "", err
	}

	return configPath, nil
}
func NewConfig(configPath string) (*Config, error) {
	config := &Config{}

	d, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("unable to read config %v", err)
	}
	if err := json.Unmarshal(d, &config); err != nil {
		log.Fatalf("unable to read config %v", err)
	}

	return config, nil
}
func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}
