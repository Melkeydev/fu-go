<!-- markdownlint-disable first-line-h1 no-inline-html -->

![logo](assets/fugo.png)

# FU-GO - Go Uninstaller

Are you angry and want to get your frustrations out? Well why not take it out on the Go programming language!

A blazingly fast way to remove Go from your system.

## Features

- Uninstalls all versions of Go on your machine!
- Increases likelihood of you not shipping.

## Installation

### 🚀 Install

```bash
go install github.com/melkeydev/fu-go@latest
```

Then in a new terminal run:

```bash
fu-go
```

to launch the TUI.

## 🛡️ Safety First

Fu-Go implements several safety measures:

- Requires typing "yes" to confirm deletion
- Performs permission checks before attempting deletion
- Displays clear warnings about the consequences
- Fails gracefully if it doesn't have necessary permissions

## 🧩 How It Works

- **Detection** - Fu-Go scans common installation locations based on your operating system.
- **Display** - Shows all found Go installations with their version information.
- **Confirmation** - Asks for explicit confirmation before proceeding.
- **Removal** - Systematically removes all Go-related directories.
- **Completion** - Notifies you when the process is complete.

## 🤝 Contributing

Contributions are welcome! Feel free to:

- Report bugs
- Suggest features
- Submit pull requests

## 🤝 Stay Single
