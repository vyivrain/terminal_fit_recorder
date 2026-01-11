# Terminal Fit Recorder

A command-line tool for tracking and managing your fitness workouts with AI-powered workout generation using Ollama.

## Features

- **Interactive workout logging** - Save strength and cardio workouts with detailed exercise information
- **AI-powered workout generation** - Get personalized workout suggestions based on your history
- **Workout management** - View, edit, and delete your workout history
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

1. **Initialize the database** (first time only):
```bash
terminal_fit_recorder exercise init
```

2. **Save your first workout**:
```bash
terminal_fit_recorder exercise save
```

3. **View your workouts**:
```bash
terminal_fit_recorder exercise last  # View most recent workout
terminal_fit_recorder exercise all   # View all workouts
```

## Commands

### `exercise init`
Initialize the database in `~/.terminal_fit_recorder/exercises.db`. Required before using any other commands.

```bash
terminal_fit_recorder exercise init
```

### `exercise generate`
Generate AI-powered workout suggestions based on your workout history using Ollama.

```bash
terminal_fit_recorder exercise generate <number_of_exercises>
```

The AI will analyze your previous workouts and suggest:
- Balanced workout types (alternating strength/cardio)
- Exercises targeting the same muscle groups as your routine
- Appropriate weights and reps based on your history

Generated workouts can be saved as "planned" for future sessions.

### `exercise help`
Display help information with all available commands.

```bash
terminal_fit_recorder exercise help
```

## Configuration

The tool uses environment variables for Ollama configuration:

```bash
export TERMINAL_FIT_RECORDER_OLLAMA_HOST="http://192.168.1.39:11434"

export TERMINAL_FIT_RECORDER_OLLAMA_MODEL="qwen3-coder:480b-cloud"

export TERMINAL_FIT_RECORDER_OLLAMA_PROMPT="Your custom prompt here"
```

These're default ollama host and ollama model. The prompt default is also made, but you can customize it.

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
