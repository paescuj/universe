package main

import (
  "os"
  "fmt"
  "sort"
  "context"
  "text/template"
  "io/ioutil"
  "encoding/json"
  "net/http"
  "github.com/google/go-github/v33/github"
  "github.com/gregjones/httpcache"
  "github.com/gregjones/httpcache/diskcache"
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

func FetchUniverse(username string) ([]*github.Repository, error) {
  context := context.Background()
  cachedir := ".cache"
	cache := diskcache.New(cachedir)
	transport := httpcache.NewTransport(cache)
	httpClient := &http.Client{Transport: transport}
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
  // Load templates
  template, err := template.ParseGlob("templates/*.gomd")
  if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

  // Fetch universe
  //universe, err := FetchUniverse("paescuj")
  universe, err := TestFetchUniverse()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

  // Init UNSORTED.md file
  unsortedFile, err := os.Create("UNSORTED.md")
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
  }

  // Write UNSORTED.md file
  err = template.ExecuteTemplate(unsortedFile, "UNSORTED.gomd", universe)
  unsortedFile.Close()
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    return
  }

  // Init README.md file
  readmeFile, err := os.Create("README.md")
  if err != nil {
		fmt.Printf("Error: %v\n", err)
    return
  }

  // Sort universe
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
}
