package editor

import (
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultEditor(t *testing.T) {
	t.Run("provided editor", func(t *testing.T) {
		got := DefaultEditor("vim")
		assert.Equal(t, "vim", got)
	})

	for _, env := range []string{"VISUAL", "EDITOR"} {
		t.Run(env, func(t *testing.T) {
			orig := os.Getenv(env)
			defer os.Setenv(env, orig)

			require.NoError(t, os.Unsetenv(env))
			require.NoError(t, os.Setenv(env, "code"))

			got := DefaultEditor("")
			assert.Equal(t, "code", got)
		})
	}

	t.Run("default", func(t *testing.T) {
		// Backup and defer restore
		for _, env := range []string{"VISUAL", "EDITOR"} {
			orig := os.Getenv(env)
			defer os.Setenv(env, orig)
			require.NoError(t, os.Unsetenv(env))
		}

		got := DefaultEditor("")

		var want string
		switch runtime.GOOS {
		case "windows":
			want = "notepad"
		default:
			want = "nano"
		}

		assert.Equal(t, want, got)
	})
}

func TestEdit(t *testing.T) {
	if os.Getenv("MANUAL_TEST") != "1" {
		t.Skip("Skipping manual test; set MANUAL_TEST=1 to run.")
	}

	editor := "nano"
	initial := "Hello, edit this content and save."

	content, err := Edit(editor, initial)
	require.NoError(t, err)

	require.NotEqual(t, initial, content, "Expected the user to change the content")
	require.True(t, strings.Contains(content, "edit"), "Expected the edited content to contain 'edit'")
}
