package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"

	"github.com/charmbracelet/huh"
	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
)

type UI interface {
	// SelectBoilerplate asks the user to choose a boilerplate.
	SelectBoilerplate(boilerplates map[string]*Boilerplate) (string, error)

	// Select asks the user to choose among a list of possible choices.
	Select(prompt string, choices []string) (string, error)

	// Prompt expects an answer from the user.
	Prompt(prompt string) (string, error)
}

type FuzzyConfig struct{}

type Fuzzy struct {
	config FuzzyConfig
}

func NewFuzzy(config FuzzyConfig) (UI, error) {
	return &Fuzzy{
		config: config,
	}, nil
}

func (u *Fuzzy) SelectBoilerplate(boilerplates map[string]*Boilerplate) (string, error) {
	// Sort the boilerplates by usage count
	var bps []*Boilerplate
	for _, bp := range boilerplates {
		bps = append(bps, bp)
	}
	sort.Slice(bps, func(i, j int) bool {
		return bps[i].Count > bps[j].Count
	})

	idx, err := fuzzyfinder.Find(
		bps,
		func(i int) string {
			return fmt.Sprintf("%5d %s", bps[i].Count, bps[i].Name)
		},
		fuzzyfinder.WithPreviewWindow(func(i, _, _ int) string {
			if i == -1 {
				return ""
			}
			return bps[i].Value
		}),
	)
	if err != nil {
		return "", err
	}

	return bps[idx].Name, nil
}

func (u *Fuzzy) Select(prompt string, choices []string) (string, error) {
	idx, err := fuzzyfinder.Find(
		choices,
		func(i int) string {
			return choices[i]
		},
	)
	if err != nil {
		return "", err
	}
	return choices[idx], nil
}

// Prompt expects an answer from the user.
func (u *Fuzzy) Prompt(prompt string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s> ", prompt)
	return reader.ReadString('\n')
}

type TermUI struct{}

func NewTermUI() UI {
	return &TermUI{}
}

func (u *TermUI) SelectBoilerplate(boilerplates map[string]*Boilerplate) (string, error) {
	var bps []*Boilerplate
	for _, bp := range boilerplates {
		bps = append(bps, bp)
	}
	sort.Slice(bps, func(i, j int) bool {
		return bps[i].Count > bps[j].Count
	})

	var opts []huh.Option[string]
	for _, bp := range bps {
		opts = append(opts, huh.NewOption[string](fmt.Sprintf("%4d %s", bp.Count, bp.Name), bp.Name))
	}

	var name string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Boilerplate to expland").
				Options(opts...).
				Value(&name),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return name, nil
}

func (u *TermUI) Select(prompt string, choices []string) (string, error) {
	var value string

	err := huh.NewSelect[string]().
		Title("Boilerplate to expland").
		Options(huh.NewOptions[string](choices...)...).
		Value(&value).
		Run()
	if err != nil {
		return "", err
	}

	return value, nil
}

func (u *TermUI) Prompt(prompt string) (string, error) {
	var value string

	err := huh.NewInput().
		Title(prompt).
		Value(&value).
		Run()
	if err != nil {
		return "", err
	}

	return value, nil
}
