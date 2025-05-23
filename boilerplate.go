package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// TODO
type Config struct {
	// RemoteCSV
	// Color config
}

type Boilerplate struct {
	Name  string
	Value string
	Count int
}

// BoilerplateManager manages a collection of boilerplates.
type BoilerplateManager struct {
	config       Config
	boilerplates map[string]*Boilerplate
	ui           UI
}

var variableRe = regexp.MustCompile(`(\[\[([a-zA-Z0-9_]+)\]\]|{{[^}]+}})`)

func NewBoilerplateManager() (*BoilerplateManager, error) {
	// Locate the boilerplate file
	// home := os.Getenv("HOME")
	// boilerplatePath := filepath.Join(home, "ezbp", "boilerplates.csv")
	boilerplatePath := "boilerplates.csv"

	// Read boilerplates
	boilerplates, err := readBoilerplatesFromPath(boilerplatePath)
	if err != nil {
		return nil, err
	}

	return &BoilerplateManager{
		// TODO config
		boilerplates: boilerplates,
		ui:           NewTermUI(),
	}, nil
}

func (bm *BoilerplateManager) SelectBoilerplate() (string, error) {
	return bm.ui.SelectBoilerplate(bm.boilerplates)
}

func (bm *BoilerplateManager) Expand(name string) (string, error) {
	bp, found := bm.boilerplates[name]
	if !found {
		return "", fmt.Errorf("unknown boilerplate %q", name)
	}

	before := bp.Value
	var after string
	var err error
	for {
		after, err = bm.expandFirst(before)
		if err != nil {
			return "", err
		}

		if before == after {
			break
		}
		before = after
	}

	// Update the boilerplate score count
	bp.Count++

	return after, nil
}

func (bm *BoilerplateManager) expandFirst(value string) (string, error) {
	// Find the first variable part to expand
	loc := variableRe.FindStringIndex(value)
	if loc == nil {
		// No variable part
		return value, nil
	}

	// Variable part found
	start, end := loc[0], loc[1]
	outerValue := value[start:end]
	innerValue := value[start+2 : end-2]
	var replacement string
	if value[start] == '[' {
		// Substitution by another boilerplate
		bp, found := bm.boilerplates[outerValue]
		if !found {
			return "", fmt.Errorf("unknown referenced boilerplate %q", innerValue)
		}
		replacement = bp.Value
	} else {
		// User prompt
		// It can consist in asking the user an open question {{prompt}}
		// Or in asking a question with a fixed set of answers {{prompt|a|b|c}}
		if idx := strings.IndexRune(innerValue, '|'); idx >= 0 {
			elements := strings.Split(innerValue, "|")
			choice, err := bm.ui.Select(elements[0], elements[1:])
			if err != nil {
				return "", err
			}
			replacement = choice
		} else {
			input, err := bm.ui.Prompt(innerValue)
			if err != nil {
				return "", err
			}
			replacement = input
		}
	}

	return value[:start] + replacement + value[end:], nil
}

// TODO
// readBoilerplatesFromPath reads a CSV file that contains the boilerplates and their usage count.
func readBoilerplatesFromPath(path string) (map[string]*Boilerplate, error) {
	// Open the file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("unable to open %q: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)

	boilerplates := make(map[string]*Boilerplate)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) != 2 && len(record) != 3 {
			return nil, fmt.Errorf("expecting 2 or 3 fields in CSV record, got %d", len(record))
		}

		name := record[0]
		value := record[1]

		var count int
		if len(record) == 3 {

			count, err = strconv.Atoi(record[2])
			if err != nil {
				return nil, fmt.Errorf("unable to convert count %s: %w", record[2], err)
			}
		}

		boilerplates[record[0]] = &Boilerplate{
			Name:  name,
			Value: value,
			Count: count,
		}
	}

	return boilerplates, nil
}

// TODO
func writeBoilerplatesToPath(path string, boilerplates map[string]*Boilerplate) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("unable to create file %q: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)

	for _, bp := range boilerplates {
		record := []string{bp.Name, bp.Value, fmt.Sprintf("%d", bp.Count)}
		if err := w.Write(record); err != nil {
			return err
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return err
	}

	return nil
}
