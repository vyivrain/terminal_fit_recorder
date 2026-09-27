# Terminal Fit Recorder

A keyboard-first terminal application for tracking workouts and generating plans with your existing OpenCode, Codex, or Claude login.

## Features

- **Full-screen terminal UI** - Browse, create, inspect, and delete workouts without memorizing commands
- **Interactive workout logging** - Save strength and cardio workouts with detailed exercise information
- **AI-powered workout generation** - Generate and safely refine workout suggestions using an installed AI CLI
- **Workout management** - View, edit, and delete your workout history
- **Separate profiles** - Keep each person's workouts and exercise suggestions isolated
- **Smart exercise tracking** - Autocomplete for exercise names and duration tracking for cardio
- **Local database storage** - All data stored securely in `~/.terminal_fit_recorder/exercises.db`

## Installation

### Download

Download the latest release for your platform from the [Releases page](https://github.com/yourusername/terminal_fit_recorder/releases).

Available platforms:
- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

### Install

```bash
# Extract the archive
tar -xzf terminal_fit_recorder-<version>-<platform>.tar.gz

# Make it executable
chmod +x terminal_fit_recorder

# Move to your PATH (optional)
sudo mv terminal_fit_recorder /usr/local/bin/
```

### macOS Security Warning

If you see "Apple could not verify terminal_fit_recorder is free of malware", you can bypass this warning:

**Method 1: Command line**
```bash
# Remove the quarantine attribute
xattr -d com.apple.quarantine terminal_fit_recorder
```

**Method 2: System Settings**
1. Go to System Settings → Privacy & Security
2. Scroll down to find the blocked app message
3. Click "Open Anyway"

## Quick Start

1. Make sure at least one supported AI CLI is installed and signed in: `opencode`, `codex`, or `claude`.

2. Start the application:

```bash
terminal_fit_recorder
```

The TUI creates or upgrades `~/.terminal_fit_recorder/exercises.db` automatically. Existing command-based workflows remain available.

### TUI keys

- `enter` — open the selected workout
- `n` — create a completed workout
- `i` — import free-form workout notes with AI
- `d` — delete the selected workout after confirmation
- `g` — generate a workout with the selected AI model
- `a` — toggle between upcoming workouts and all dates
- `f` — filter workouts by planned or completed status
- `p` — manage profiles
- `m` — switch AI models
- `/` — search the current list
- `q` — quit

The workout list starts in upcoming mode and sorts the nearest workout first. Date scope and status filters are independent, and the status modal also lets you clear the status filter.

The AI preview uses `e` to manually edit the current workout in a prefilled exercise table, `a` to request a natural-language AI modification, `s` to save as planned, and `esc` to discard. AI modification prompts are constrained to exercise selection and exercise parameters; unrelated requests are refused.

The exercise table includes name, weight (kg), repetitions, sets, duration (minutes), and distance (metres). Use `↑`/`↓` to select a row and `←`/`→` or `tab` to select a column, then `enter` to edit its value. `enter` accepts a cell; `esc` cancels that cell. `ctrl+s` applies the edits to the preview, including the cell currently being edited. Outside a cell, `esc` cancels the table edits. Nothing is written to the database until you save the preview with `s`. AI modifications use the latest manually edited preview. IDs and timestamps are managed by the database.

The file importer accepts UTF-8 text without a required format, including Ukrainian notes. Each weekday section becomes a separate planned workout on the next matching future date. Exercise names are translated to English, explicit metrics are preserved, and missing weights are reused only when the active profile has a similar weighted exercise in its completed history. Bodyweight and distance exercises remain unweighted. Scroll an individual workout with `↑`/`↓`, browse imported workouts with `←`/`→`, edit the current workout with `e` or ask AI with `a`, then press `s` to save the full batch.

Profile management supports create, rename, confirmed delete, and setting the default profile. Existing data belongs to `mine`; every new profile starts empty.

The model picker includes:

- `opencode-go/glm-5.3` through OpenCode
- `gpt-5.6-sol` through Codex
- the rolling `opus` alias through Claude

The application reuses each CLI's existing login. OpenCode requests run through an isolated temporary workout-only agent with tools disabled; it does not modify global OpenCode configuration. If a CLI, login, model, or provider endpoint is unavailable, the TUI displays the returned error in a modal.

## Commands

### `exercise init`
Explicitly initialize the database in `~/.terminal_fit_recorder/exercises.db` for command-only use. Launching the TUI initializes it automatically; otherwise, run this once before the other explicit commands.

```bash
terminal_fit_recorder exercise init
```

### `exercise generate`
Use the legacy command-based Ollama generator. The TUI's `g` workflow instead uses the selected OpenCode, Codex, or Claude CLI and its existing login.

```bash
terminal_fit_recorder exercise generate <number_of_exercises>
```

The AI will analyze your previous workouts and suggest:
- Balanced workout types (alternating strength/cardio)
- Exercises targeting the same muscle groups as your routine
- Appropriate weights and reps based on your history

Generated workouts can be saved as "planned" for future sessions.

### `exercise save --file`
Import one or more completed workouts from a JSON file. The app previews every parsed workout first, then asks whether to approve, discard, or edit the import. Approve saves the full batch; discard saves nothing. Edit is reserved for a future interactive editor.

```bash
terminal_fit_recorder exercise save --file workouts.json
terminal_fit_recorder exercise save workouts.json
```

Supported JSON shapes:

```json
[
  {
    "date": "2026-01-05",
    "type": "strength",
    "exercises": [
      { "name": "Bench Press", "weight": 80, "reps": 10, "sets": 3 }
    ]
  }
]
```

The file can also use `{ "workouts": [...] }` or contain a single workout object.

### `exercise help`
Display help information with all available commands.

```bash
terminal_fit_recorder exercise help
```

### Profiles

The active profile controls every workout and exercise command, including autocomplete and AI workout generation. Profile names are case-insensitively unique.

```bash
terminal_fit_recorder profile create <name>  # Create an empty profile and select it
terminal_fit_recorder profile use <name>     # Select an existing profile
terminal_fit_recorder profile list           # List profiles; * marks the active one
terminal_fit_recorder profile help           # Display all commands
```

## Configuration

The legacy `exercise generate` command uses these environment variables for Ollama configuration. They do not affect TUI generation:

```bash
export TERMINAL_FIT_RECORDER_OLLAMA_HOST="http://192.168.1.39:11434"

export TERMINAL_FIT_RECORDER_OLLAMA_MODEL="qwen3-coder:480b-cloud"

export TERMINAL_FIT_RECORDER_OLLAMA_PROMPT="Your custom prompt here"
```

The application provides defaults for the Ollama host, model, and prompt; set these only when you want to override them.

## Database Location

All workout data is stored in: `~/.terminal_fit_recorder/exercises.db`

## Workout Types

- **Strength**: Weightlifting and resistance exercises
- **Cardio**: Aerobic exercises (running, cycling, etc.)

## Exercise Tracking

For each exercise, you can track:
- Name
- Weight
- Repetitions
- Sets
- Duration (automatically prompted for cardio exercises)
- Distance (for specific type of exercises)
