package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/xeonx/timeago"
)

var Key string
var Reader *bufio.Reader
var programDetails ProgramsWithStats
var programs Programs

func main() {
	Reader = bufio.NewReader(os.Stdin)
	requestAPIKey()
	getProgramSummary()
	for {
		prompt()
	}
}

func prompt() {
	fmt.Println(`type "new" for a new program, "list", "popularity", "logs" or "quit":`)
	var command string
	fmt.Scanln(&command)
	switch command {
	case "quit":
		os.Exit(0)
	case "ls":
		listPrograms()
	case "list":
		listPrograms()
	case "new":
		createNewProgram()
	case "logs":
		viewLogs()
	case "popularity":
		viewPopularity()
	}
}

func viewLogs() {
	if len(programs) == 0 || programs == nil {
		getAndSortProgramDetails()
	}

	var from string
	var to string
	fmt.Println("This is an expensive operation, so it's going to take some time. FYI. Type quit to return back.")
	fmt.Println("Enter a from date (inclusive) like this 2016-10-29 or 2016-09-05 (yyyy-mm-dd):")
	fmt.Scanln(&from)
	if from == "quit" {
		return
	}
	fmt.Println("Enter a to date (inclusive) like this 2016-10-29 or 2016-09-05 (yyyy-mm-dd):")
	fmt.Scanln(&to)
	if to == "quit" {
		return
	}

	fromArray := strings.Split(from, "-")
	toArray := strings.Split(to, "-")

	fromYear, err := strconv.Atoi(fromArray[0])
	fromMonth, err := strconv.Atoi(fromArray[1])
	fromDay, err := strconv.Atoi(fromArray[2])

	toYear, err := strconv.Atoi(toArray[0])
	toMonth, err := strconv.Atoi(toArray[1])
	toDay, err := strconv.Atoi(toArray[2])

	if err != nil {
		fmt.Println("Invalid dates")
		return
	}

	fromDate := time.Date(fromYear, time.Month(fromMonth), fromDay, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(toYear, time.Month(toMonth), toDay, 0, 0, 0, 0, time.UTC)

	var episodesToAccess []Episode
	for _, program := range programDetails {
		for _, episode := range program.Episodes {
			if episode.Pubdate.After(fromDate) && episode.Pubdate.Before(toDate) {
				episodesToAccess = append(episodesToAccess, episode)
			}
		}
	}

	csvWriter := csv.NewWriter(os.Stdout)

	for _, episode := range episodesToAccess {
		getJson("http://api.teal.cool/episodes/"+episode.ID, &episode)
		for _, track := range episode.Tracks {
			log := Log{"WJRH", track.LogTime, track.Title, track.Artist}
			if log.LogTime == nil {
				log.LogTime = episode.Pubdate
			}
			array := []string{log.Station, log.LogTime.String(), log.Title, log.Artist}
			csvWriter.Write(array)
			csvWriter.Flush()
		}
	}

}

type Log struct {
	Station string
	LogTime *time.Time
	Title   string
	Artist  string
}

func viewPopularity() {
	if len(programs) == 0 || programs == nil {
		getAndSortProgramDetails()
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Program", "Total Listens"})

	for _, program := range programDetails {
		for _, episode := range program.Episodes {
			program.TotalListens += episode.Hits
		}
		table.Append([]string{program.Name, strconv.Itoa(program.TotalListens)})
	}

	fmt.Println()
	table.Render()

}

func createNewProgram() {
	var name, author, response, ownersRaw string
	var ownersList, owners []string
	fmt.Print(`Program name: `)
	name, _ = Reader.ReadString('\n')
	name = strings.TrimSpace(name)

	fmt.Print(`Authors (DJs): `)
	author, _ = Reader.ReadString('\n')
	author = strings.TrimSpace(author)

	fmt.Print("Owner emails: \n(separate multiple people with spaces! This program will add @lafayette.edu if you don't add it)\n")
	ownersRaw, _ = Reader.ReadString('\n')
	ownersList = strings.Split(ownersRaw, " ")
	owners = []string{}
	for _, owner := range ownersList {
		owner := strings.TrimSpace(owner)
		if !strings.Contains(owner, "@") {
			owner = owner + "@lafayette.edu"
		}
		owners = append(owners, owner)
	}
	fmt.Println("\nIs everything correct, are you sure you'd like to create this program? (yes or no)")
	for response != "yes" && response != "no" {
		fmt.Scanln(&response)
	}
	program := Program{
		Author:        author,
		Name:          name,
		Owners:        owners,
		Organizations: []string{"wjrh"},
		Stream:        "http://wjrh.org:8000/WJRHlow",
	}
	if response == "yes" {
		postJson("https://api.teal.cool/programs/", program)
	}
}

func listPrograms() {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Latest Ep", "Program", "Author"})

	if len(programs) == 0 || programs == nil {
		getAndSortProgramDetails()
	}

	timeago.English.Max = 170 * time.Hour

	for _, program := range programDetails {
		table.Append([]string{getPrettyTimeAgo(program.LastPubdate), program.Name, program.Author})
	}
	fmt.Println()
	table.Render()
}

func getAndSortProgramDetails() {
	programs = getProgramSummary()
	fmt.Print("Loading programs")
	var programDetailsWithoutEpisodes ProgramsWithStats

	for _, program := range programs {
		var programDetail ProgramWithStats
		getJson("http://api.teal.cool/programs/"+program.Shortname, &programDetail)
		fmt.Print(".")
		getLatestEpsiodeDateRelative(&programDetail)
		if programDetail.LastPubdate != nil {
			programDetails = append(programDetails, programDetail)
		} else {
			programDetailsWithoutEpisodes = append(programDetailsWithoutEpisodes, programDetail)
		}
	}

	sort.Sort(programDetails)
	programDetails = append(programDetails, programDetailsWithoutEpisodes...)
}

func getPrettyTimeAgo(time *time.Time) string {
	if time == nil {
		return "never"
	}
	return timeago.English.Format(*time)
}

func getLatestEpsiodeDateRelative(program *ProgramWithStats) {
	if len(program.Episodes) == 0 {
		return
	}
	sort.Sort(program.Episodes)
	numEpisodes := len(program.Episodes)
	lastEpTime := *program.Episodes[numEpisodes-1].Pubdate
	program.LastPubdate = &lastEpTime
}

func requestAPIKey() {
	text := `Teal Program Manager Tool

You can manage WJRH programs in bulk with this application.
You will now need to enter your API key.
Your API key can be found at https://api.teal.cool/key after logging into Teal.
`
	fmt.Println(text)
	for len(Key) != 88 {
		fmt.Println("Enter your API key:")
		_, err := fmt.Scanln(&Key)
		if err != nil {
			panic("There was an error while reading your API key")
		}
	}
}

func getProgramSummary() Programs {
	fmt.Println("loading programs...")
	var programs Programs
	getJson("http://api.teal.cool/organizations/wjrh", &programs)
	fmt.Println(len(programs), "programs found on WJRH\n")
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

type ProgramWithStats struct {
	Program
	LastPubdate  *time.Time
	TotalListens int
}

type Program struct {
	Active           bool     `json:"active,omitempty"`
	Author           string   `json:"author,omitempty"`
	Copyright        string   `json:"copyright,omitempty"`
	CoverImage       string   `json:"cover_image,omitempty"`
	Description      string   `json:"description,omitempty"`
	Explicit         bool     `json:"explicit,omitempty"`
	Image            string   `json:"image,omitempty"`
	ItunesCategories []string `json:"itunes_categories,omitempty"`
	Language         string   `json:"language,omitempty"`
	Name             string   `json:"name,omitempty"`
	Organizations    []string `json:"organizations,omitempty"`
	Owners           []string `json:"owners,omitempty"`
	RedirectURL      string   `json:"redirect_url,omitempty"`
	ScheduledTime    string   `json:"scheduled_time,omitempty"`
	Shortname        string   `json:"shortname,omitempty"`
	Stream           string   `json:"stream,omitempty"`
	Subtitle         string   `json:"subtitle,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	ID               string   `json:"id",omitempty`
	Episodes         Episodes `json:"episodes",omitempty`
}

type Episode struct {
	AudioURL    string     `json:"audio_url"`
	Delay       int        `json:"delay"`
	Description string     `json:"description"`
	EndTime     *time.Time `json:"end_time"`
	Explicit    bool       `json:"explicit"`
	GUID        string     `json:"guid"`
	Hits        int        `json:"hits"`
	Image       string     `json:"image"`
	Length      string     `json:"length"`
	Name        string     `json:"name"`
	Pubdate     *time.Time `json:"pubdate"`
	StartTime   *time.Time `json:"start_time"`
	Type        string     `json:"type"`
	ID          string     `json:"id"`
	Tracks      []Track    `json:"tracks"`
}

type Track struct {
	Artist  string     `json:"artist"`
	LogTime *time.Time `json:"log_time"`
	Mbid    string     `json:"mbid"`
	Title   string     `json:"title"`
	ID      string     `json:"id"`
}

type Episodes []Episode

func (x Episodes) Len() int           { return len(x) }
func (x Episodes) Swap(i, j int)      { x[i], x[j] = x[j], x[i] }
func (x Episodes) Less(i, j int) bool { return x[i].Pubdate.Before(*x[j].Pubdate) }

type ProgramsWithStats []ProgramWithStats

func (x ProgramsWithStats) Len() int      { return len(x) }
func (x ProgramsWithStats) Swap(i, j int) { x[i], x[j] = x[j], x[i] }
func (x ProgramsWithStats) Less(j, i int) bool {
	return x[i].LastPubdate.Before(*x[j].LastPubdate)
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
		panic("Problem decoding some piece of JSON " + url + err.Error())
	}
}

func postJson(url string, target interface{}) {
	client := &http.Client{}
	thing, err := json.Marshal(target)
	if err != nil {
		panic("Problem encoding some piece of JSON target " + err.Error())
	}
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(thing))
	req.Header.Set("teal-api-key", Key)
	res, err := client.Do(req)
	if err != nil {
		panic("Problem with communicating with " + url + "\n" + err.Error())
	}

	if res.StatusCode == 200 {
		fmt.Println("Successful!")
	} else {
		fmt.Println("There was a problem, did not get 200 OK from Teal.")
		buf := new(bytes.Buffer)
		buf.ReadFrom(res.Body)
		s := buf.String()
		fmt.Println(s)
	}

}
