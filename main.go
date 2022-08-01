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
	"sync"
	"time"

	twitter "github.com/g8rswimmer/go-twitter/v2"
)

var config *Config
var nsfw = regexp.MustCompile(`((?i)(NSFW|ONLYFANS|MODEL))|18\+|🔞`)
var nsfwLink = regexp.MustCompile(`((?i)(ONLYFANS|FANSLY|FANHOUSE|CASHAPP))`)

// 18+, NSWF, OF(case sensitive), onlyfans, 18+ emoji
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
	var ctx context.Context
	var nextToken string
	following, err := FollowingLookup(ctx, person[0].ID, client, nextToken)
	exposed := Expose(following)
	fmt.Printf("%s is following %d NSFW twitter accounts.\n The accounts are: ", person[0].UserName, len(exposed))

	for i := 0; i < len(exposed); i++ {
		if i == len(exposed)-1 {
			fmt.Println(exposed[i].UserName)
		} else {
			fmt.Printf(exposed[i].UserName + ", ")
		}
	}
}
func getActualUrl(url string) (string, error) {
	resp, err := http.Head(url)
	if err != nil {
		return "", err
	}

	finalURL := resp.Request.URL.String()

	return finalURL, nil
}

func Expose(users []*twitter.UserObj) []*twitter.UserObj {
	exposedUsers := []*twitter.UserObj{}
	mtx := new(sync.Mutex)
	var wg sync.WaitGroup
	for i := 0; i < len(users); i++ {
		wg.Add(1)
		go func(user *twitter.UserObj) {
			if nsfw.MatchString(user.Description) {
				mtx.Lock()
				exposedUsers = append(exposedUsers, user)
				mtx.Unlock()
				wg.Done()
				return
			}
			url, err := getActualUrl(user.URL)
			if err != nil {
				wg.Done()
				return
			}
			if nsfwLink.MatchString(url) {
				mtx.Lock()
				exposedUsers = append(exposedUsers, user)
				mtx.Unlock()
			}
			wg.Done()
		}(users[i])
	}
	wg.Wait()
	return exposedUsers
}
func UsernameLookup(ctx context.Context, username string, client *twitter.Client) ([]*twitter.UserObj, error) {

	opts := twitter.UserLookupOpts{}
	userResponse, err := client.UserNameLookup(context.Background(), strings.Split(username, ","), opts)

	if rateLimit, has := twitter.RateLimitFromError(err); has && rateLimit.Remaining == 0 {
		time.Sleep(time.Until(rateLimit.Reset.Time()))
		userResponse, err = client.UserNameLookup(context.Background(), strings.Split(username, ","), opts)
		return userResponse.Raw.Users, err
	}
	return userResponse.Raw.Users, err
}
func FollowingLookup(ctx context.Context, id string, client *twitter.Client, nextToken string) ([]*twitter.UserObj, error) {
	opts := twitter.UserFollowingLookupOpts{
		UserFields:      []twitter.UserField{twitter.UserFieldDescription, twitter.UserFieldURL},
		PaginationToken: nextToken,
		MaxResults:      1000,
	}

	userResponse, err := client.UserFollowingLookup(context.Background(), id, opts)
	if rateLimit, has := twitter.RateLimitFromError(err); has && rateLimit.Remaining == 0 {
		fmt.Printf("Too many request, program continuing at: %v\n\n", rateLimit.Reset.Time())
		time.Sleep(time.Until(rateLimit.Reset.Time()))
		fmt.Println("Continuing... (this may take a while)")
		userResponse, err = client.UserFollowingLookup(context.Background(), id, opts)
		return userResponse.Raw.Users, err
	}
	if userResponse.Meta.NextToken != "" {
		rec, rec_err := FollowingLookup(ctx, id, client, userResponse.Meta.NextToken)
		if rec_err != nil {
			return []*twitter.UserObj{}, err
		}
		userResponse.Raw.Users = append(userResponse.Raw.Users, rec...)
		return userResponse.Raw.Users, err
	}
	return userResponse.Raw.Users, err
}

func (a authorize) Add(req *http.Request) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", a.Token))
}
func (m User) IsEmpty() bool {
	return reflect.DeepEqual(User{}, m)
}
func input(x []string, err error) []string {
	if err != nil {
		return x
	}
	var d string

	n, err := fmt.Scanf("%s", &d)
	if n == 1 {
		x = append(x, d)
	}
	return input(x, err)
}
func UserSearch(c *twitter.Client) []*twitter.UserObj {
	fmt.Printf("Enter the username of the person you're searching (must be 1 word): ")
	people := input([]string{}, nil)
	for len(people) != 1 || !regexp.MustCompile(`^[a-zA-Z0-9_]*$`).MatchString(people[0]) {
		fmt.Printf("username entered incorrectly, try again: ")
		people = input([]string{}, nil)
	}
	fmt.Println("Searching up " + people[0] + "... (this may take a while)")
	var ctx context.Context
	resp, err := UsernameLookup(ctx, people[0], c)
	if err != nil {
		return []*twitter.UserObj{}
	}

	return resp
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
