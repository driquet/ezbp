# ezbp

**A command-line tool for quick text generation from reusable boilerplates.**

`ezbp` (easy boilerplate) simplifies the process of inserting frequently used text snippets, especially those that require dynamic user input or choices among predefined options. It allows you to create a library of text templates and quickly expand them, with the final output automatically copied to your clipboard.

## Features

*   **Define Reusable Text Boilerplates:** Store and manage your common text snippets.
*   **Dynamic User Prompts:** Use `{{prompt_text}}` to ask for free-form user input during expansion.
*   **Multiple Choice Selections:** Use `{{prompt_text|choice1|choice2|...}}` to offer a list of options.
*   **Include Other Boilerplates:** Embed existing boilerplates within others using `[[other_boilerplate_name]]`.
*   **Fuzzy Search:** Quickly find and select boilerplates using an interactive fuzzy search interface.
*   **Usage Counting & Sorting:** `ezbp` tracks how often each boilerplate is used and sorts them by frequency for easier access.
*   **Configuration via TOML:** Customize `ezbp` behavior through a simple configuration file (`~/.config/ezbp/config.toml`).
*   **SQLite Backend:** Boilerplates are stored in an SQLite database for robust and efficient data management.
*   **Multiple UI Options:** Supports a terminal-based UI and an integration with [Rofi](https://github.com/davatorium/rofi) for a keyboard-driven experience.
*   **Clipboard Integration:** The final expanded text is automatically copied to your system clipboard.

## Installation

You can install `ezbp` using `go install`:

```bash
go install github.com/driquet/ezbp@latest
```

Alternatively, you can clone the repository and build from source:

```bash
git clone https://github.com/USERNAME/ezbp.git
cd ezbp
go build
# You can then move the compiled binary to a directory in your PATH
```

## Configuration

`ezbp` uses a configuration file located at `~/.config/ezbp/config.toml`.
The `ezbp` directory and the `config.toml` file are automatically created with default settings on the first run if they don't already exist.

### Configuration Options

`ezbp` is configured using a TOML file. The main options are:

*   **`database_path`**:
    *   **Purpose:** Specifies the full path to your SQLite database file where boilerplates are stored.
    *   **Default:** `~/.config/ezbp/ezbp.db`
    *   **Example:** `database_path = "/path/to/your/custom/ezbp.db"`

*   **`default_ui`**:
    *   **Purpose:** Sets the default user interface to use if the `--ui` command-line flag is not provided.
    *   **Valid values:** `"terminal"`, `"rofi"`
    *   **Default:** `"terminal"`
    *   **Example:** `default_ui = "terminal"`

*   **`[RofiUI]` table**:
    *   **Purpose:** Configures settings specific to the Rofi user interface. These settings are applied *if* Rofi is selected as the UI (either via the `--ui rofi` flag or `default_ui = "rofi"` in the config).
    *   **Options:**
        *   `path` (string): Command or full path to the Rofi executable.
            *   Default: `"rofi"` (assumes `rofi` is in your system's PATH).
        *   `theme` (string, optional): Specifies a Rofi theme file to use (e.g., `"solarized"`, `"dracula"`). If empty, Rofi's default theme or theme specified in Rofi's own configuration will be used.
            *   Default: `""`
        *   `select_args` (array of strings, optional): Extra command-line arguments to pass to Rofi when it's used for selection dialogs (e.g., choosing a boilerplate, selecting from multiple choice options).
            *   Default: `[]` (empty list)
            *   Example: `select_args = ["-i", "-p", "Choose:"]` (for case-insensitive search and a custom prompt)
        *   `input_args` (array of strings, optional): Extra command-line arguments to pass to Rofi when it's used for free-form text input dialogs.
            *   Default: `[]` (empty list)
            *   Example: `input_args = ["-password"]` (for password-style input where characters are hidden)
    *   **Example `config.toml` snippet:**
        ```toml
        # Full path to the SQLite database file.
        database_path = "~/.config/ezbp/ezbp.db"

        # Default UI to use ("terminal" or "rofi").
        # Overridden by the --ui command-line flag.
        default_ui = "terminal"

        # Rofi User Interface settings
        # These settings are used if default_ui = "rofi" or --ui=rofi is specified.
        [RofiUI]
          # Path to the Rofi executable.
          path = "rofi" # Or specify full path, e.g., "/usr/bin/rofi"
          # Optional: Specify a Rofi theme file.
          # theme = "your_rofi_theme"
          # Extra arguments for Rofi selection dialogs.
          # select_args = ["-i", "-no-custom", "-kb-accept-entry", "Return,KP_Enter,Control+m"]
          # Extra arguments for Rofi input dialogs.
          # input_args = ["-historic-all", "false"]
        ```
        When `ezbp` creates a default `config.toml` for the first time, it will include these settings with explanatory comments.

### UI Selection Precedence

The user interface used by `ezbp` is determined in the following order:

1.  **`--ui` command-line flag:** If you use `ezbp boilerplate expand --ui rofi` or `ezbp boilerplate expand --ui terminal`, this choice takes highest precedence.
2.  **`default_ui` in `config.toml`:** If the `--ui` flag is not provided, `ezbp` will use the UI specified in the `default_ui` field of your configuration file.
3.  **Application Default:** If neither the `--ui` flag is used nor the `default_ui` field is set or valid in the config, `ezbp` defaults to the "terminal" UI.

## Boilerplate Storage (SQLite Database)

`ezbp` stores all your defined boilerplates in an SQLite database file.

*   **Location:** The path to this database file is specified by the `database_path` option in your `config.toml`.
    *   **Default:** `~/.config/ezbp/ezbp.db`
*   **Automatic Creation:** The database file and its directory (e.g., `~/.config/ezbp/`) are automatically created by `ezbp` on its first run if they don't already exist.
*   **Schema:** For those interested, the `boilerplates` table in the database has the following key fields:
    *   `name` (TEXT, UNIQUE): The unique identifier for the boilerplate.
    *   `value` (TEXT): The template string, which can include placeholders.
    *   `count` (INTEGER): The number of times the boilerplate has been used. `ezbp` updates this automatically.
    *   Other fields include `id` (PRIMARY KEY), `created_at`, and `updated_at`.
*   **Management:** Currently, adding, editing, or removing boilerplates directly via CLI commands is a planned future improvement. For now, you would need to use an SQLite database browser or editor to manage boilerplates if you need to make changes outside of the `ezbp` application's normal usage (which only updates the count).

## Usage

The primary command to use `ezbp` is:

```bash
ezbp boilerplate expand [--ui <value>]
```

*   `--ui <value>` (optional): Specify the user interface. Valid values are `"terminal"` or `"rofi"`. This flag overrides the `default_ui` setting in the configuration file.
    *   Example: `ezbp boilerplate expand --ui rofi`

**Process:**

1.  You will be presented with an interactive list of your defined boilerplates, sorted by usage count (most used first). You can type to fuzzy search through this list.
2.  Select the desired boilerplate.
3.  `ezbp` will process the selected boilerplate's template string.
4.  If the template contains any placeholders:
    *   For `{{prompt_text}}`, you'll be prompted to enter text.
    *   For `{{prompt_text|choice1|choice2}}`, you'll be prompted to select one of the choices.
    *   `[[other_boilerplate_name]]` will be replaced by the content of the referenced boilerplate (which itself might be expanded if it contains placeholders).
5.  Once all placeholders are resolved, the final expanded text is automatically copied to your clipboard.

## Boilerplate Syntax Reference

Use these placeholders within the `value` field of your boilerplates (stored in the SQLite database):

*   **`{{prompt_text}}`**: Prompts the user for free-form text input. The `prompt_text` will be displayed to the user.
    *   Example: `Meeting notes for {{meeting_subject}}.`

*   **`{{prompt_text|choice1|choice2|...}}`**: Prompts the user to select one option from a list. The `prompt_text` is displayed, followed by the choices.
    *   Example: `Project status: {{Select status|On Track|Delayed|Completed}}`

*   **`[[boilerplate_name]]`**: Includes the expanded content of another boilerplate. `boilerplate_name` must match the `name` field of an existing boilerplate in the database.
    *   Example: If you have a boilerplate named `signature` with the value `Thanks, {{Your Name}}`, you can use `[[signature]]` in another boilerplate.

## Contributing

Issues and Pull Requests are welcome! If you find a bug or have a feature request, please open an issue on the GitHub repository.

## License

Distributed under the MIT License. See the `LICENSE` file for more information.

## Future Improvements

Here are some ideas for potential future enhancements to `ezbp`:

*   **Remote Boilerplate Storage:** Allow syncing or storing boilerplates in remote locations such as a Git repository, GitHub Gist, or other cloud storage services.
*   **CLI for Boilerplate Management:** Introduce dedicated CLI commands for managing boilerplates directly (e.g., `add`, `edit`, `rm`, `list`). This is a high-priority next step now that the SQLite backend is in place.
    *   `ezbp boilerplate add`: Interactively add a new boilerplate.
    *   `ezbp boilerplate edit <name>`: Edit an existing boilerplate.
    *   `ezbp boilerplate rm <name>`: Remove a boilerplate.
    *   `ezbp boilerplate list`: List all available boilerplates with details.
*   **Enhanced Boilerplate Syntax/Logic:**
    *   **Conditionals:** Add support for basic conditional logic within boilerplate templates (e.g., if-then-else based on prompt inputs).
    *   **Predefined Variables:** Introduce system variables (e.g., current date/time, username) that can be used in templates.
    *   **Text Transformations:** Allow simple text transformations on user inputs (e.g., case changes, default values if input is empty).
*   **UI/UX Enhancements:**
    *   **Color Configuration:** Allow users to customize UI colors through the `config.toml` file (this applies mainly to the terminal UI).
    *   **Better Previews:** Improve the preview window in the fuzzy finder (if re-enabled) or terminal UI to better represent complex boilerplates.
*   **Shell Completions:** Generate shell completion scripts for Bash, Zsh, Fish, etc., to improve CLI usability.
*   **"Dry Run" or Preview Mode:** Add an option to preview the fully expanded boilerplate in the terminal before copying it to the clipboard.
