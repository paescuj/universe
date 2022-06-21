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
  "github.com/google/go-github/v37/github"
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

func TestFetchUniverse() ([]*github.Repository, []string, error) {
  var universe []*github.Repository
  file, err := ioutil.ReadFile("test.json")
  if err != nil {
    return nil, nil, err
  }
  err = json.Unmarshal([]byte(file), &universe)
  if err != nil {
    return nil, nil, err
  }
  ignoredRepos := []string{"repo/1", "repo/2"}
  return universe, ignoredRepos, nil
}

func FetchUniverse(username string, useCache bool) ([]*github.Repository, []string, error) {
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
  var ignoredRepos []string
  for {
    stars, response, err := githubClient.Activity.ListStarred(context, username, githubOpts)
    if err != nil {
      return nil, nil, err
    }
    for _, star := range stars {
      repo := star.GetRepository()
      // Check if repo still exists
      response, _ := http.Get(repo.GetHTMLURL())
      if response.StatusCode == http.StatusNotFound {
        // Add to list of ignored repos
        ignoredRepos = append(ignoredRepos, repo.GetFullName())
      } else {
        // Add repo to universe
        universe = append(universe, repo)
      }
    }
    if response.NextPage == 0 {
      break
    }
    githubOpts.Page = response.NextPage
  }
  return universe, ignoredRepos, nil
}

func main() {
  // Initialize spinner & cache
  spinnerSet := spinner.CharSets[9]
  spinnerSpeed := 100*time.Millisecond
  useCache := true
  if (os.Getenv("CI") == "true") {
    spinnerSet = []string{"+"}
    spinnerSpeed = 5*time.Second
    useCache = false
  }
  spinner := spinner.New(spinnerSet, spinnerSpeed, spinner.WithWriter(os.Stderr))

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
  //var _ = useCache; universe, ignoredRepos, err := TestFetchUniverse()
  universe, ignoredRepos, err := FetchUniverse("paescuj", useCache)
  time.Sleep(1 * time.Second)
  if ignoredRepos != nil {
    message := "The following repos have been ignored:\n"
    for _, repo := range ignoredRepos {
      message += fmt.Sprintf("- %s\n", repo)
    }
    spinner.FinalMSG = message
  }
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
  }
  time.Sleep(1 * time.Second)
  spinner.Stop()
  spinner.FinalMSG = ""

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
