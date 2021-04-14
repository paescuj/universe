package main

import (
  "os"
  "fmt"
  "sort"
  "context"
  "time"
  "text/template"
  "io/ioutil"
  "encoding/json"
  "net/http"
  "github.com/google/go-github/v35/github"
  "github.com/gregjones/httpcache"
  "github.com/gregjones/httpcache/diskcache"
  "github.com/briandowns/spinner"
)

type SortedUniverse struct {
  LivingStars []*github.Repository
  DeadStars   []*github.Repository
}

func (s SortedUniverse) Count() int {
  return len(s.LivingStars)+len(s.DeadStars)
}

func SaveTestData(universe []*github.Repository) (error) {
  file, _ := json.MarshalIndent(universe, "", " ")
  err := ioutil.WriteFile("test.json", file, 0644)
  if err != nil {
    return err
  }
  return nil
}

func TestFetchUniverse() ([]*github.Repository, error) {
  var universe []*github.Repository
	file, err := ioutil.ReadFile("test.json")
  if err != nil {
	  return nil, err
  }
  err = json.Unmarshal([]byte(file), &universe)
  if err != nil {
	  return nil, err
  }
  return universe, nil
}

func FetchUniverse(username string, useCache bool) ([]*github.Repository, error) {
  context := context.Background()
  httpClient := &http.Client{
    Timeout: time.Second * 20,
  }

  if useCache {
    cachedir := ".cache"
    cache := diskcache.New(cachedir)
    transport := httpcache.NewTransport(cache)
    httpClient.Transport = transport
  }

  githubClient := github.NewClient(httpClient)
  githubOpts := &github.ActivityListStarredOptions{
    ListOptions: github.ListOptions{PerPage: 100},
  }

  var universe []*github.Repository
  for {
    stars, response, err := githubClient.Activity.ListStarred(context, username, githubOpts)
    if err != nil {
      return nil, err
    }
	  for _, star := range stars {
      universe = append(universe, star.GetRepository())
	  }
    if response.NextPage == 0 {
      break
    }
    githubOpts.Page = response.NextPage
  }
  return universe, nil
}

func main() {
  // Initialize spinner
  spinner := spinner.New(spinner.CharSets[9], 100*time.Millisecond)

  // Load templates
  spinner.Suffix = " Loading templates..."
  spinner.Start()
  template, err := template.ParseGlob("templates/*.gomd")
  if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
  spinner.Stop()

  // Fetch universe
  spinner.Suffix = " Fetching universe..."
  spinner.Start()
  universe, err := FetchUniverse("paescuj", false)
  //universe, err := TestFetchUniverse()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
  spinner.Stop()

  // Init HISTORY.md file
  spinner.Suffix = " Creating HISTORY.md file..."
  spinner.Start()
  historyFile, err := os.Create("HISTORY.md")
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
  }

  // Write HISTORY.md file
  err = template.ExecuteTemplate(historyFile, "HISTORY.gomd", universe)
  historyFile.Close()
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
  }
  spinner.Stop()

  // Init README.md file
  spinner.Suffix = " Creating README.md file..."
  spinner.Start()
  readmeFile, err := os.Create("README.md")
  if err != nil {
		fmt.Printf("Error: %v\n", err)
    return
  }

  // Sort universe (by date & type)
  sort.Slice(universe, func(i, j int) bool {
    return universe[i].GetPushedAt().Time.After(universe[j].GetPushedAt().Time)
  })
  sortedUniverse := SortedUniverse{}
  for _, star := range universe {
    if !star.GetArchived() {
      sortedUniverse.LivingStars = append(sortedUniverse.LivingStars, star)
    } else {
      sortedUniverse.DeadStars = append(sortedUniverse.DeadStars, star)
    }
  }

  // Write README.md file
  err = template.ExecuteTemplate(readmeFile, "README.gomd", sortedUniverse)
  readmeFile.Close()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
  spinner.Stop()
}
