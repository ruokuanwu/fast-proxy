package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

type Rule struct {
	ID     string `json:"id"`
	Domain string `json:"domain"`
	Target string `json:"target"`
}

type State struct {
	Rules []Rule `json:"rules"`
}

func (s *State) EnsureIDs() (bool, error) {
	changed := false
	for i := range s.Rules {
		if s.Rules[i].ID != "" {
			continue
		}

		id, err := s.NewRuleID()
		if err != nil {
			return false, err
		}
		s.Rules[i].ID = id
		changed = true
	}
	return changed, nil
}

func (s *State) NewRuleID() (string, error) {
	for {
		id, err := generateRuleID()
		if err != nil {
			return "", err
		}
		if !s.HasID(id) {
			return id, nil
		}
	}
}

func (s *State) HasID(id string) bool {
	for _, rule := range s.Rules {
		if rule.ID == id {
			return true
		}
	}
	return false
}

func (s *State) Upsert(rule Rule) error {
	for i, existing := range s.Rules {
		if existing.Domain == rule.Domain {
			if rule.ID == "" {
				rule.ID = existing.ID
			}
			s.Rules[i] = rule
			s.Sort()
			return nil
		}
	}
	if rule.ID == "" {
		id, err := s.NewRuleID()
		if err != nil {
			return err
		}
		rule.ID = id
	}
	s.Rules = append(s.Rules, rule)
	s.Sort()
	return nil
}

func (s *State) RemoveByID(id string) (Rule, bool) {
	for i, rule := range s.Rules {
		if rule.ID == id {
			s.Rules = append(s.Rules[:i], s.Rules[i+1:]...)
			return rule, true
		}
	}
	return Rule{}, false
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
	changed, err := state.EnsureIDs()
	if err != nil {
		return State{}, err
	}
	state.Sort()
	if changed {
		if err := s.Save(state); err != nil {
			return State{}, err
		}
	}
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

func generateRuleID() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
