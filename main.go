package main

import (
  "os"
  "fmt"
  "sort"
  "context"
  "time"
  "strings"
  "text/template"
  "io/ioutil"
  "encoding/json"
  "net/http"
  "github.com/google/go-github/v56/github"
  "github.com/gregjones/httpcache"
  "github.com/gregjones/httpcache/diskcache"
  "github.com/briandowns/spinner"
  "golang.org/x/oauth2"
  "github.com/shurcooL/githubv4"
)

type CategorizedUniverse struct {
  LivingStars []*github.Repository
  DeadStars   []*github.Repository
}
func (universe CategorizedUniverse) Count() int {
  return len(universe.LivingStars)+len(universe.DeadStars)
}

func saveTestData(universe []*github.Repository) error {
  file, _ := json.MarshalIndent(universe, "", " ")
  err := ioutil.WriteFile("test.json", file, 0644)
  if err != nil {
    return err
  }
  return nil
}

func testFetchUniverse(universe *[]*github.Repository) error {
  file, err := ioutil.ReadFile("test.json")
  if err != nil {
    return err
  }
  err = json.Unmarshal([]byte(file), &universe)
  if err != nil {
    return err
  }
  return nil
}

func fetchUniverse(universe *[]*github.Repository, username string, useCache bool) error {
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

  for {
    stars, response, err := githubClient.Activity.ListStarred(context, username, githubOpts)
    if err != nil {
      return err
    }
    for _, star := range stars {
      *universe = append(*universe, star.GetRepository())
    }
    if response.NextPage == 0 {
      break
    }
    githubOpts.Page = response.NextPage
  }
  return nil
}

func filterNonExistingStars(universe *[]*github.Repository, githubToken string) error {
  src := oauth2.StaticTokenSource(
    &oauth2.Token{AccessToken: githubToken},
  )
  httpClient := oauth2.NewClient(context.Background(), src)
  client := githubv4.NewClient(httpClient)

  type NonExistingStar struct {
    Index int
    Name  string
  }
  var nonExistingStars []NonExistingStar

  // Split into chunks with length of 100
  // (GitHub limitation for GraphQL API)
  chunkSize := 100
  for i := 0; i < len(*universe); i += chunkSize {
    end := i + chunkSize
    if end > len(*universe) {
      end = len(*universe)
    }

    // Prepare query
    var query struct {
      Nodes []struct {
        Repository struct {
          Name string
        } `graphql:"... on Repository"`
      } `graphql:"nodes(ids: $id)"`
    }
    var ids []githubv4.ID
    // Get and append node ids
    for _, star := range (*universe)[i:end] {
      ids = append(ids, githubv4.ID(star.GetNodeID()))
    }
    variables := map[string]interface{}{
      "id": ids,
    }
    err := client.Query(context.Background(), &query, variables)
    if err != nil && !strings.HasPrefix(err.Error(), "Could not resolve to a node with the global id of") {
      return err
    }
    // If no name is returned the repo no longer exists
    for index, node := range query.Nodes {
      if node.Repository.Name == "" {
        nonExistingStars = append(nonExistingStars, NonExistingStar{i+index, (*universe)[i+index].GetFullName()})
      }
    }
  }

  // Filter out all non existing stars
  if len(nonExistingStars) > 0 {
    fmt.Println("The following stars have been ignored because they no longer seem to exist:")
    for i := len(nonExistingStars) - 1; i >= 0; i-- {
      nonExistingStar := nonExistingStars[i]
      fmt.Printf("- %s (%d)\n", nonExistingStar.Name, nonExistingStar.Index)
      *universe = append((*universe)[:nonExistingStar.Index], (*universe)[nonExistingStar.Index+1:]...)
    }
  }

  return nil
}

func main() {
  // -- Initialize spinner --
  spinnerSet := spinner.CharSets[9]
  spinnerSpeed := 100*time.Millisecond
  spinner := spinner.New(spinnerSet, spinnerSpeed, spinner.WithWriter(os.Stderr))

  // -- Load templates --
  spinner.Suffix = " Loading templates..."
  spinner.Start()

  template, err := template.ParseGlob("templates/*.gomd")
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
  }

  spinner.Stop()

  // -- Fetch universe --
  spinner.Suffix = " Fetching universe..."
  spinner.Start()

  username, usernamePresent := os.LookupEnv("GITHUB_REPOSITORY_OWNER")
  if !usernamePresent {
    // Fallback for username
    username = "paescuj"
  }
  useCache := true
  if os.Getenv("CI") == "true" {
    useCache = false
  }
  var universe []*github.Repository
  //var _ = useCache; var _ = username; err = testFetchUniverse(&universe)
  err = fetchUniverse(&universe, username, useCache)
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
  }

  // Check whether the stars really exist and filter out non existing ones
  // (it happened that the GitHub API has returned stars even for repos which no longer exist)
  if os.Getenv("CHECK_STARS") == "true" {
    // Only proceed if GitHub token is present
    githubToken, githubTokenPresent := os.LookupEnv("GITHUB_TOKEN")
    if len(universe) > 0 && githubTokenPresent {
      err := filterNonExistingStars(&universe, githubToken)
      if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
      }
    }
  }

  spinner.Stop()

  // -- Create HISTORY.md file --
  spinner.Suffix = " Creating HISTORY.md file..."
  spinner.Start()

  // Init HISTORY.md file
  historyFile, err := os.Create("HISTORY.md")
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
  }

  // Write HISTORY.md file
  err = template.ExecuteTemplate(historyFile, "HISTORY.gomd", universe)
  historyFile.Close()
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
  }

  spinner.Stop()

  // -- Create README.md file --
  spinner.Suffix = " Creating README.md file..."
  spinner.Start()

  // Init README.md file
  readmeFile, err := os.Create("README.md")
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
  }

  // Sort universe by date
  sort.Slice(universe, func(i, j int) bool {
    return universe[i].GetPushedAt().Time.After(universe[j].GetPushedAt().Time)
  })

  // Categorize universe by type
  categorizedUniverse := CategorizedUniverse{}
  for _, star := range universe {
    if !star.GetArchived() {
      categorizedUniverse.LivingStars = append(categorizedUniverse.LivingStars, star)
    } else {
      categorizedUniverse.DeadStars = append(categorizedUniverse.DeadStars, star)
    }
  }

  // Write README.md file
  err = template.ExecuteTemplate(readmeFile, "README.gomd", categorizedUniverse)
  readmeFile.Close()
  if err != nil {
    fmt.Printf("Error: %v\n", err)
    os.Exit(1)
  }

  spinner.Stop()
}
