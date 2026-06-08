package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

type Rule struct {
	Domain string `json:"domain"`
	Target string `json:"target"`
}

type State struct {
	Rules []Rule `json:"rules"`
}

func (s *State) Upsert(rule Rule) {
	for i, existing := range s.Rules {
		if existing.Domain == rule.Domain {
			s.Rules[i] = rule
			s.Sort()
			return
		}
	}
	s.Rules = append(s.Rules, rule)
	s.Sort()
}

func (s *State) Remove(domain string) bool {
	for i, rule := range s.Rules {
		if rule.Domain == domain {
			s.Rules = append(s.Rules[:i], s.Rules[i+1:]...)
			return true
		}
	}
	return false
}

func (s *State) Sort() {
	sort.Slice(s.Rules, func(i, j int) bool {
		return s.Rules[i].Domain < s.Rules[j].Domain
	})
}

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Load() (State, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, err
	}
	if len(data) == 0 {
		return State{}, nil
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	state.Sort()
	return state, nil
}

func (s *Store) Save(state State) error {
	state.Sort()
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
