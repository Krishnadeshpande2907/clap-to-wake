# Clap-to-Wake

> A Go application that monitors your target applications and wakes your system with a clap detection using audio input.

## Table of Contents
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Building the Executable](#building-the-executable)
- [Running the Application](#running-the-application)
- [Configuration](#configuration)
- [Project Structure](#project-structure)
- [Dependencies](#dependencies)
- [Development](#development)
- [Testing](#testing)
- [Known Issues](#known-issues)
- [Contributing](#contributing)
- [License](#license)

---

## Overview

Clap-to-Wake is a creative GO utility that creates an interactive terminal UI experience for system control. It monitors whether specific applications (code, Chrome, Cursor, VLC, etc.) are running, listens for audio signals through your microphone, detects clap patterns when sound levels exceed a configurable threshold, and automatically wakes the system by simulating a mouse nudge and keyboard input.

The application features a beautiful terminal user interface powered by Bubble Tea, displaying real-time status updates, microphone input levels as a progress bar, and information about active monitoring states. It's designed to be lightweight when idle (low energy mode) and activate microphone monitoring when target applications are detected.

## Prerequisites

- **Go version**: 1.25.6 or later (as specified in `go.mod`)
- **OS compatibility**: 
  - Linux (with ALSA/PulseAudio audio system)
  - macOS (with CoreAudio)
  - Windows (with WASAPI)
  - Note: `robotgo` requires system-level permissions for mouse/keyboard control
- **Audio hardware**: Working microphone/audio input device
- **System permissions**: 
  - Audio input access (microphone permissions)
  - Keyboard and mouse control permissions (especially on Linux/macOS with security restrictions)
- **External tools**: 
  - GCC or Clang C compiler (required by CGomusic library dependencies like malgo and robotgo)
  - `pkg-config` (on Linux for audio header files)
- **Required environment setup**:
  - On Linux: ALSA development libraries (`libasound2-dev` on Ubuntu/Debian)
  - On macOS: Xcode Command Line Tools
  - On Windows: MinGW or MSVC toolchain

## Installation

1. **Clone the repository**:
```bash
git clone https://github.com/Krishnadeshpande2907/clap-to-wake.git
```

2. **Navigate into the directory** (already done in step 1).
```bash
cd clap-to-wake
```

3. **Download and verify dependencies**:
```bash
go mod download
go mod verify
```

4. **Install build dependencies** (if needed):
   
   **Ubuntu/Debian**:
   ```bash
   sudo apt-get install -y libasound2-dev libx11-dev libxtst-dev libxinerama-dev
   ```
   
   **macOS**:
   ```bash
   # Usually pre-installed with Xcode Command Line Tools
   xcode-select --install
   ```
   
   **Windows**:
   - Install MinGW-w64 or Microsoft Visual C++ Build Tools

## Building the Executable

### Standard build

```bash
go build -o clap-to-wake ./main.go
```

This creates a binary named `clap-to-wake` (or `clap-to-wake.exe` on Windows) in the current directory.

### Cross-compilation

Build for different operating systems and architectures:

**Linux (AMD64)**:
```bash
GOOS=linux GOARCH=amd64 go build -o clap-to-wake-linux ./main.go
```

**macOS (ARM64 - Apple Silicon)**:
```bash
GOOS=darwin GOARCH=arm64 go build -o clap-to-wake-darwin-arm64 ./main.go
```

**macOS (AMD64 - Intel)**:
```bash
GOOS=darwin GOARCH=amd64 go build -o clap-to-wake-darwin-amd64 ./main.go
```

**Windows (AMD64)**:
```bash
GOOS=windows GOARCH=amd64 go build -o clap-to-wake.exe ./main.go
```

**Note**: Cross-compilation of CGO-dependent projects (like this one with malgo and robotgo) may not work across all OS/architecture combinations. Native compilation on the target platform is recommended.

### Build flags

The current codebase does not define custom build flags. For production builds, you may consider injecting version information:

```bash
go build -ldflags="-s -w" -o clap-to-wake ./main.go  # Strip debug symbols for smaller size
```

### Output

The compiled binary is placed in the current working directory with the name `clap-to-wake` (or `clap-to-wake.exe` on Windows).

## Running the Application

### Standard execution

```bash
./clap-to-wake
```

### Using `go run` (development)

```bash
go run ./main.go
```

### CLI flags and options

The application currently does **not** accept command-line flags. All configuration is hardcoded in the initial model (see [Configuration](#configuration) section). 

### User interaction

Once running, the application displays a real-time TUI with:
- Current monitoring state (Watcher, Idle Monitor, Active Listener, or Waking System)
- Status messages
- Microphone input level (visual progress bar when listening)
- List of monitored applications
- Timeout setting
- Help text: **Press 'q' to quit the application**

## Configuration

All configuration is currently hardcoded in the `initialModel()` function in `main.go`. To modify behavior, edit the following fields in the code:

| Configuration | Default Value | Type | Description |
|---|---|---|---|
| TargetApps | `["code", "chrome", "cursor", "browser", "vlc", "msedge"]` | `[]string` | List of application names to monitor (case-insensitive substring match) |
| Threshold | `0.25` | `float32` | Audio sensitivity threshold for clap detection (0.0 = silent, 1.0 = loud). Based on RMS (Root Mean Square) calculation of audio samples. |
| IdleLimit | `300` | `uint32` | Time in seconds before idle check expires (currently disabled). Default is 5 minutes. |
| Sample Rate | `44100` | `uint32` (hardcoded) | Audio capture sample rate in Hz. Can be modified in `listenForAudio()`. |
| Audio Format | `FormatF32` | `enum` (hardcoded) | Audio format is 32-bit float. Defined in `listenForAudio()`. |
| Audio Channels | `1` | `uint8` (hardcoded) | Mono input. Can be modified in `listenForAudio()`. |
| Listen Timeout | `5` | `seconds` (hardcoded) | How long to listen for a clap before returning to app monitoring (5 seconds). Set in `listenForAudio()`. |

### Example: Modifying configuration

Edit `main.go` in the `initialModel()` function:

```go
return model{
    cfg: Config{
        TargetApps: []string{"vscode", "firefox", "slack"},  // Your apps
        Threshold:  0.35,  // More sensitive
        IdleLimit:  600,   // 10 minutes instead of 5
    },
    // ...
}
```

Then rebuild:
```bash
go build -o clap-to-wake ./main.go
```

## Project Structure

```
clap-to-wake/
├── main.go              # Single source file containing all logic
│                        # - TUI model and state management (Bubble Tea)
│                        # - Core monitoring loop (watches for target apps)
│                        # - Audio capture and clap detection (malgo)
│                        # - System wake logic via mouse/keyboard (robotgo)
│                        # - Audio byte-to-float32 conversion helpers
├── go.mod               # Module definition (module: clap-to-wake, Go 1.25.6)
├── go.sum               # Dependency checksums and versions
├── .git/                # Git repository metadata
├── .gitignore           # Git ignore rules
└── clap-to-wake         # Compiled binary (generated after build)
```

### Code organization within main.go:

1. **Package & Imports** - Standard library and third-party dependencies
2. **Constants & Configuration** - Application states (StateWatchingApps, StateWaitingIdle, etc.) and Config struct
3. **Styling** - Bubble Tea/Lipgloss style definitions for the TUI
4. **Bubble Tea Model** - statusUpdateMsg, model struct, initialModel(), Init(), Update(), View()
5. **Core Logic** - monitorLoop(), listenForAudio() with audio processing
6. **Utilities** - byteSliceToFloat32() for audio data conversion
7. **Main Entry Point** - main() function

## Dependencies

### Direct dependencies (from go.mod):

| Package | Version | Purpose |
|---|---|---|
| `github.com/charmbracelet/bubbles` | v1.0.0 | Provides reusable UI components: progress bars and spinners for the TUI |
| `github.com/charmbracelet/bubbletea` | v1.3.10 | Core TUI framework; manages application state, updates, and rendering |
| `github.com/charmbracelet/lipgloss` | v1.1.0 | Styling library for terminal UI components; defines colors and text formatting |
| `github.com/gen2brain/malgo` | v0.11.24 | Audio capture library using miniaudio backend; handles microphone input and audio data buffering |
| `github.com/go-vgo/robotgo` | v1.0.2 | System automation library; provides mouse movement and keyboard input control for wake-up |
| `github.com/shirou/gopsutil/v3` | v3.24.5 | System/process utilities; used to list running processes for application monitoring |

### Key transitive dependencies (selection):

- **charmbracelet/x/ansi**, **charmbracelet/x/term**, **charmbracelet/harmonica** - Bubble Tea ecosystem utilities for terminal handling and math
- **golang.org/x/sys**, **golang.org/x/text**, **golang.org/x/image** - Standard Go extended libraries for system calls, text encoding, and image processing
- **muesli/ansi**, **muesli/termenv** - Terminal rendering and ANSI sequence helpers
- **rivo/uniseg** - Unicode segmentation for proper text width calculation in TUI

### Why each dependency:

- **Bubble Tea** - Simplifies building interactive terminal UIs with event-driven architecture
- **Charmbracelet suite** - Professional-quality terminal styling and components
- **malgo** - Cross-platform, lightweight audio capture without heavy dependencies
- **robotgo** - Unified API for mouse/keyboard control across Linux, macOS, Windows
- **gopsutil** - Lightweight process and system information gathering without external commands

## Development

### Running in development mode

As the application is a single-file Go program with no hot-reload capability, development requires rebuilding:

```bash
# Development build with full debug info
go build -o clap-to-wake ./main.go

# Run the development binary
./clap-to-wake
```

Alternatively, run directly without building:

```bash
go run ./main.go
```

### Debugging audio issues

If the application fails to detect audio or microphone:

1. **Check microphone permissions**:
   - Linux: Ensure user is in `audio` group: `groups $USER | grep audio`
   - macOS: Check System Preferences > Security & Privacy > Microphone
   - Windows: Check Settings > Privacy & Security > Microphone

2. **List available audio devices** (platform-specific):
   - Linux: `arecord -l` (ALSA) or `pactl list sources` (PulseAudio)
   - macOS: `system_profiler SPAudioDataType`
   - Windows: Settings > Sound > Advanced > Volume mixer

3. **Test with alternative audio recorder** (e.g., `sox`, `arecord`, `ffmpeg`) to verify hardware is working.

### Key code sections to modify:

- **Monitor loop interval** `monitorLoop()`: Change `time.Sleep(3 * time.Second)` to adjust polling frequency
- **Audio capture parameters** `listenForAudio()`: Adjust `SampleRate`, `Channels`, or listen timeout
- **Clap detection sensitivity**: Adjust `cfg.Threshold` in `initialModel()`
- **System wake action**: Modify the wake logic in `listenForAudio()` where `robotgo.Move()` and `robotgo.KeyTap()` are called

## Known Issues

1. **IdleTime not available in robotgo v1.0.2**:
   - The idle detection feature is currently disabled (set to `if false` in `monitorLoop()`)
   - Upgrade robotgo or implement alternative idle time detection if needed
   - TODO in code indicates this limitation

2. **No CLI flags**:
   - Configuration requires code modification and recompilation
   - Consider implementing flag parsing (e.g., `flag` package) or config file support for production use

3. **Single-threaded audio listen loop**:
   - The 5-second listen timeout is fixed in code
   - May miss claps that occur just after timeout expires
   - Consider extending timeout or implementing a continuous background listener

4. **Cross-compilation challenges**:
   - CGO dependencies (malgo, robotgo) generally require native compilation
   - Windows/Linux/macOS builds may have platform-specific audio/system call issues

---