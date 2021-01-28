package main

import (
  "github.com/google/go-github/v33/github"
)

type SortedUniverse struct {
	LivingStars []*github.Repository
	DeadStars   []*github.Repository
}

func (s SortedUniverse) Count() int {
	return len(s.LivingStars)+len(s.DeadStars)
}
