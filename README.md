# Unsplash MCP Server

> A Model Context Protocol server that provides Unsplash photo search and retrieval capabilities. This server enables LLMs to search, retrieve, and get random photos from Unsplash's extensive collection. This is a Swift implementation of the [Python version](https://github.com/hellokaton/unsplash-mcp-server), with additional features like `get_photo` and `random_photo`.

![Swift Platform](https://img.shields.io/badge/platform-macOS-lightgrey)
![License](https://img.shields.io/badge/license-MIT-blue)

## ✨ Features

* **Advanced Image Search**: Search Unsplash's extensive photo library with filters for:
  * Keyword relevance
  * Color schemes
  * Orientation options
  * Custom sorting and pagination
* **Detailed Photo Information**: Get comprehensive details about specific photos including EXIF data, location, and photographer information
* **Random Photo Selection**: Get random photos with flexible filtering options

## Available Tools

* `search_photos` - Search for photos on Unsplash
  * `query` (string, required): Search keyword
  * `page` (number, optional): Page number (1-based), default: 1
  * `perPage` (number, optional): Results per page (1-30), default: 10
  * `orderBy` (string, optional): Sort method (relevant or latest), default: "relevant"
  * `color` (string, optional): Color filter (black_and_white, black, white, yellow, orange, red, purple, magenta, green, teal, blue)
  * `orientation` (string, optional): Orientation filter (landscape, portrait, squarish)

* `get_photo` - Get detailed information about a specific photo
  * `photoId` (string, required): The photo ID to retrieve

* `random_photo` - Get one or more random photos
  * `count` (number, optional): The number of photos to return (Default: 1; Max: 30)
  * `collections` (string, optional): Public collection ID('s) to filter selection. If multiple, comma-separated
  * `topics` (string, optional): Public topic ID('s) to filter selection. If multiple, comma-separated
  * `username` (string, optional): Limit selection to a specific user
  * `query` (string, optional): Limit selection to photos matching a search term
  * `orientation` (string, optional): Filter by photo orientation. Valid values: landscape, portrait, squarish
  * `content_filter` (string, optional): Limit results by content safety. Valid values: low, high
  * `featured` (boolean, optional): Limit selection to featured photos

## Installation

### Option 1: One-Line Installation (curl)

The easiest way to install is with the one-line installer, which automatically downloads the latest version and installs it to `~/.local/bin` in your home directory:

```bash
curl -fsSL https://raw.githubusercontent.com/okooo5km/unsplash-mcp-server-swift/main/install.sh | bash
```

The installer will:

* Create `~/.local/bin` if it doesn't exist
* Add this directory to your PATH (in .zshrc or .bashrc)
* Download and install the latest version
* Make the binary executable

### Option 2: Build from Source

1. Clone the repository:

   ```bash
   git clone https://github.com/okooo5km/unsplash-mcp-server-swift.git
   cd unsplash-mcp-server-swift
   ```

2. Build the project:

   ```bash
   swift build -c release
   ```

3. Install the binary:

   ```bash
   # Install to user directory (recommended, no sudo required)
   mkdir -p ~/.local/bin
   cp $(swift build -c release --show-bin-path)/unsplash-mcp-server ~/.local/bin/
   ```

   Make sure `~/.local/bin` is in your PATH by adding to your shell configuration file:

   ```bash
   echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc  # or ~/.bashrc
   source ~/.zshrc  # or source ~/.bashrc
   ```

## Configuration

### Environment Variables

The server requires an Unsplash API access key to function. Set it in your environment:

```bash
export UNSPLASH_ACCESS_KEY="your-access-key-here"
```

* Unsplash API Access Key (register at [Unsplash Developers Portal](https://unsplash.com/developers))

### Obtain an Unsplash API Access Key

1. Go to the [Unsplash Developers Portal](https://unsplash.com/developers)
2. Sign up or log in to your Unsplash account
3. Register a new application
4. Accept the API use and guidelines
5. Fill in your application details (name, description, etc.)
6. Once registered, you'll receive your Access Key (also called Client ID)

### Configure for Claude.app

Add to your Claude settings:

```json
"mcpServers": {
  "unsplash": {
    "command": "unsplash-mcp-server"
  }
}
```

### Configure for Cursor

Add the following configuration to your Cursor editor's `settings.json`:

```json
{
  "mcpServers": {
    "unsplash": {
      "command": "unsplash-mcp-server",
      "env": {
        "UNSPLASH_ACCESS_KEY": "${YOUR_ACCESS_KEY}"
      }
    }
  }
}
```

![](screenshots/SCR-20250318-unsw.webp)

### Configure for Chatwise

![](screenshots/SCR-20250318-umwn.webp)

### Use in Chatwise

<video src="https://github.com/user-attachments/assets/ade1e290-3c95-4561-aa32-fa3a729160c4" controls></video>

## Development Requirements

* Swift 6.0 or later
* macOS 14.0 or later
* Unsplash API access key
* MCP Swift SDK 0.2.0 or later

## Version History

See GitHub Releases for version history and changelog.

### ☕️ Support the Project

If you find unsplash-mcp-server useful, please consider supporting its development:

* ⭐️ Star the project on GitHub
* 🐛 Report bugs or suggest features in the issue tracker
* 🔄 Submit pull requests to help improve the code
* 💝 Support via:

<p align="center">
  <a href="https://buymeacoffee.com/okooo5km">
    <img src="https://img.buymeacoffee.com/button-api/?text=Buy me a coffee&emoji=&slug=okooo5km&button_colour=FFDD00&font_colour=000000&font_family=Cookie&outline_colour=000000&coffee_colour=ffffff" style="border-radius: 8px;" />
  </a>
</p>

## License

unsplash-mcp-server is licensed under the MIT License. This means you are free to use, modify, and distribute the software, subject to the terms and conditions of the MIT License. For more details, please see the LICENSE file in the project repository.
