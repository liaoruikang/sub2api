# OpenAI Image Generator

Python + PySide6 desktop image generator for OpenAI-compatible image APIs.

## Features

- Configure Base URL, API Key, timeout, concurrency, and save directory.
- Use OpenAI-compatible Images API: `/v1/images/generations`.
- Use OpenAI-compatible Chat Completions API: `/v1/chat/completions`.
- Show streaming progress events for stream-capable chat endpoints.
- Generate one image task at a time or import many prompts as a queue.
- Save generated images locally by default.
- Preview results, save a copy, copy the image, copy base64, export base64 text, and open the output directory.

## Install and Run from Source

```bash
cd image-generator
python -m venv .venv
.venv/Scripts/activate
pip install -e .[dev]
python -m image_generator
```

On macOS or Linux, activate the virtual environment with:

```bash
source .venv/bin/activate
```

## Configuration

The app stores configuration in the current user's config directory, such as:

```text
%APPDATA%/ImageGenerator/config.json
```

The first run opens the settings dialog when Base URL or API Key is missing.

## API Key Security

The first version stores the API Key in a local JSON config file. Protect your operating system account and do not share the config file. The UI only displays whether an API key is configured and the final four characters.

## Windows Build

Install development dependencies, then run:

```powershell
cd image-generator
./scripts/build-windows.ps1
```

The generated executable is placed under `image-generator/dist/`.
