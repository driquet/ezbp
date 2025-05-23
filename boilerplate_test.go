package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpand(t *testing.T) {
	bm := &BoilerplateManager{
		boilerplates: map[string]Boilerplate{
			"foobar": {
				Name:  "foobar",
				Value: "the text of foobar",
			},
			"barfoo": {
				Name:  "barfoo",
				Value: "before [[foobar]] after",
			},
			"fizzbuzz": {
				Name:  "fizzbuzz",
				Value: "john [[barfoo]] doe",
			},
		},
	}

	for name, test := range map[string]struct {
		name     string
		expected string
	}{
		"no variable": {
			name:     "foobar",
			expected: "the text of foobar",
		},
		"substitution": {
			name:     "barfoo",
			expected: "before the text of foobar after",
		},
		"nested substitution": {
			name:     "fizzbuzz",
			expected: "john before the text of foobar after doe",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := bm.Expand(test.name)
			require.NoError(t, err)
			assert.Equal(t, test.expected, got)
		})
	}
}
