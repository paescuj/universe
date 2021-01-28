package main

import (
  "github.com/google/go-github/v33/github"
)

type Stars struct {
	Active   []*github.Repository
	Archived []*github.Repository
}

func (s Stars) Count() int {
	return len(s.Active)+len(s.Archived)
}
