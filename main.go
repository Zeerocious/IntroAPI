package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"reflect"
	"regexp"

	"github.com/go-resty/resty/v2"
)

var config *Config

const (
	TWITTER_API_BASE = "https://api.twitter.com"
)

type User struct {
	Data Data `json:"data"`
}
type Config struct {
	Twitter struct {
		Bearer string `json:"Bearer_Token"`
	} `json:"twitter"`
}
type Data struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Bio      string `json:"description"`
}
type Following struct {
	Data []Data `json:"data"`
	Meta struct {
		Count     int    `json:"result_count"`
		NextToken string `json:"next_token"`
	} `json:"meta"`
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
	Person := UserSearch()
	for Person.IsEmpty() {
		fmt.Printf("User was not found\n\n")
		Person = UserSearch()
	}
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+config.Twitter.Bearer).
		Get(TWITTER_API_BASE + "/2/users/" + Person.Data.ID + "/following?user.fields=description")
	var following Following
	json.Unmarshal(resp.Body(), &following)
	fmt.Println(following.Data[3].Bio)
	//fmt.Println(string(resp.Body()))
	fmt.Println(following.Meta)
}
func (m User) IsEmpty() bool {
	return reflect.DeepEqual(User{}, m)
}
func UserSearch() User {
	var Person string
	fmt.Printf("Enter the username of the person you're searching (must be 1 word): ")
	fmt.Scan(&Person)
	for !regexp.MustCompile(`^[a-zA-Z]*$`).MatchString(Person) {

		fmt.Printf("username entered incorrectly, try again: ")
		fmt.Scan(&Person)
	}
	fmt.Println("Searching up " + Person)
	client := resty.New()
	resp, err := client.R().
		SetHeader("Authorization", "Bearer "+config.Twitter.Bearer).
		Get(TWITTER_API_BASE + "/2/users/by/username/" + Person)
	if err != nil {
		fmt.Println(err)
	}
	var PersonInfo User
	json.Unmarshal(resp.Body(), &PersonInfo)
	return PersonInfo
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
