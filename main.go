package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	twitter "github.com/g8rswimmer/go-twitter/v2"
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
type authorize struct {
	Token string
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
	client := &twitter.Client{
		Authorizer: authorize{
			Token: config.Twitter.Bearer,
		},
		Client: http.DefaultClient,
		Host:   "https://api.twitter.com",
	}

	person := UserSearch(client)
	for len(person) == 0 {
		fmt.Printf("User not found\n\n")
		person = UserSearch(client)
	}
	fmt.Println(person)
}
func UsernameLookup(ctx context.Context, username string, client *twitter.Client) (*twitter.UserLookupResponse, error) {

	opts := twitter.UserLookupOpts{}
	userResponse, err := client.UserNameLookup(context.Background(), strings.Split(username, ","), opts)

	if rateLimit, has := twitter.RateLimitFromError(err); has && rateLimit.Remaining == 0 {
		time.Sleep(time.Until(rateLimit.Reset.Time()))
		return client.UserNameLookup(context.Background(), strings.Split(username, ","), opts)
	}
	return userResponse, err
}

// change max result to 1000
// implement paragration
// 18+, NSWF, OF(case sensitive), Onlyfans
func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}
func (m User) IsEmpty() bool {
	return reflect.DeepEqual(User{}, m)
}
func UserSearch(c *twitter.Client) []*twitter.UserObj {
	var Person string
	fmt.Printf("Enter the username of the person you're searching (must be 1 word): ")
	fmt.Scan(&Person) // THIS THING IS TAKING MULTIPLE ARGUMENTS, AND IF THE FIRST ONE DOESNT COMPILE, IT WILL TAKE THE ONE AFTER. EXAMPLE: ";L;L;LJ Zeerocious", it will error out but still use Zeerocious
	for !regexp.MustCompile(`^[a-zA-Z]*$`).MatchString(Person) {

		fmt.Printf("username entered incorrectly, try again: ")
		fmt.Scan(&Person)
	}
	fmt.Println("Searching up " + Person)
	var ctx context.Context
	resp, err := UsernameLookup(ctx, Person, c)
	if err != nil {
		return []*twitter.UserObj{}
	}

	return resp.Raw.Users
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
