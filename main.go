package main

import (
  "os"
  "fmt"
  "context"
  "text/template"
  "io/ioutil"
  "encoding/json"
  "github.com/google/go-github/v33/github"
)

func fetchUniverseTest() ([]*github.Repository, error) {
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

func fetchUniverse(username string) ([]*github.Repository, error) {
  ctx := context.Background()
  client := github.NewClient(nil)
  opt := &github.ActivityListStarredOptions{
    Sort: "updated",
    ListOptions: github.ListOptions{PerPage: 100},
  }

  var universe []*github.Repository
  for {
    stars, resp, err := client.Activity.ListStarred(ctx, username, opt)
    if err != nil {
      return nil, err
    }
	  for _, star := range stars {
      universe = append(universe, star.GetRepository())
	  }
    if resp.NextPage == 0 {
      break
    }
    opt.Page = resp.NextPage
  }
  return universe, nil
}

func main() {
  tpl, err := template.ParseFiles("template.md")
  if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

  universe, err := fetchUniverse("paescuj")
  //universe, err := fetchUniverseTest()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

  file, err := os.Create("README.md")
  if err != nil {
		fmt.Printf("Error: %v\n", err)
    return
  }

  tmp := Stars{}
  for _, v := range universe {
    if !v.GetArchived()  {
      tmp.Active = append(tmp.Active, v)
    }
  }

  err = tpl.Execute(file, tmp)
  file.Close()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
}
