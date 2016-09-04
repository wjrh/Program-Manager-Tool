package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

var Key string
var Reader *bufio.Reader

func main() {
	Reader = bufio.NewReader(os.Stdin)
	requestAPIKey()
	programs := getProgramSummary()
	fmt.Println(len(programs), "programs found on WJRH")
	cont := true
	for cont {
		cont = prompt()
	}
}

func prompt() bool {
	fmt.Println("hello")
	command, _ := Reader.ReadString('\n')
	fmt.Println(command)
	return true
}

func requestAPIKey() {
	text := `Teal Program Manager Tool

You can manage WJRH programs in bulk with this application.
You will now need to enter your API key.
Your API key can be found at https://api.teal.cool/key after logging into Teal.
`
	fmt.Println(text)
	for len(Key) != 89 {
		fmt.Println("Enter your API key:")
		Key, _ = Reader.ReadString('\n')
	}
}

func getProgramSummary() Programs {
	var programs Programs
	getJson("http://api.teal.cool/organizations/wjrh", &programs)
	return programs
}

type Programs []struct {
	Author      string   `json:"author"`
	Description string   `json:"description"`
	Image       string   `json:"image"`
	Name        string   `json:"name"`
	Shortname   string   `json:"shortname"`
	Subtitle    string   `json:"subtitle"`
	Tags        []string `json:"tags"`
}

type Program struct {
	Active           bool     `json:"active"`
	Author           string   `json:"author"`
	Copyright        string   `json:"copyright"`
	CoverImage       string   `json:"cover_image"`
	Description      string   `json:"description"`
	Explicit         bool     `json:"explicit"`
	Image            string   `json:"image"`
	ItunesCategories []string `json:"itunes_categories"`
	Language         string   `json:"language"`
	Name             string   `json:"name"`
	Organizations    []string `json:"organizations"`
	Owners           []string `json:"owners"`
	RedirectURL      string   `json:"redirect_url"`
	ScheduledTime    string   `json:"scheduled_time"`
	Shortname        string   `json:"shortname"`
	Stream           string   `json:"stream"`
	Subtitle         string   `json:"subtitle"`
	Tags             []string `json:"tags"`
	ID               string   `json:"id"`
	Episodes         []struct {
		AudioURL    string     `json:"audio_url"`
		Delay       int        `json:"delay"`
		Description string     `json:"description"`
		EndTime     *time.Time `json:"end_time"`
		Explicit    string     `json:"explicit"`
		GUID        string     `json:"guid"`
		Hits        int        `json:"hits"`
		Image       string     `json:"image"`
		Length      string     `json:"length"`
		Name        string     `json:"name"`
		Pubdate     *time.Time `json:"pubdate"`
		StartTime   *time.Time `json:"start_time"`
		Type        string     `json:"type"`
		ID          string     `json:"id"`
	} `json:"episodes"`
}

func getJson(url string, target interface{}) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("teal-api-key", Key)
	res, err := client.Do(req)
	if err != nil {
		panic("Problem with communicating with " + url + "\n" + err.Error())
	}
	defer res.Body.Close()
	err = json.NewDecoder(res.Body).Decode(target)
	if err != nil {
		panic("Problem decoding some piece of JSON " + err.Error())
	}
}
