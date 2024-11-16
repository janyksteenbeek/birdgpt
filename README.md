# BirdGPT

BirdGPT automatically processes invoices from your Gmail inbox and adds them to Moneybird. It uses GPT-4o to extract invoice details and can handle Dutch KVK and BTW numbers.

> USE AT OWN RISK FOR NOW: I highly recommend testing the application with a separate Moneybird administration before using it with your actual administration. The application is still in development and may contain bugs. You can create a sandbox administration for free [here](https://moneybird.com/administrations/sandboxes/new).


- Automatically monitors Gmail for new invoices
- Extracts invoice details using GPT-4o
- Creates contacts and purchase invoices in Moneybird
- Handles Dutch KVK and BTW numbers
- Automatically matches correct tax rates
- OAuth authentication for Gmail
- Configurable email label and check interval

[![GitHub release (latest by date)](https://img.shields.io/github/v/release/janyksteenbeek/birdgpt)](https://github.com/janyksteenbeek/birdgpt/releases)
[![GitHub](https://img.shields.io/github/license/janyksteenbeek/birdgpt)](LICENSE.md)
[![GitHub issues](https://img.shields.io/github/issues/janyksteenbeek/birdgpt)](https://github.com/janyksteenbeek/birdgpt/issues)

## Prerequisits

You need to request a few API keys to get started. Follow the steps below to get everything set up.

### 1. Moneybird

1. Go to [moneybird.com/user/applications/new](https://moneybird.com/user/applications/new)
2. Create a personal token
3. Note your administration ID from the URL when logged into Moneybird (format: /123456789)

### 2. Gmail

1. Go to [Google Cloud Console](https://console.cloud.google.com)
2. Create a new project
3. Enable Gmail API
4. Create OAuth 2.0 credentials
5. Download credentials and save as `credentials.json` in the project directory

### 3. OpenAI

1. Go to [platform.openai.com](https://platform.openai.com)
2. Create an API key


## Configuration

Create a `config.yaml` file in the project root. See `config.example.yaml` for all available options.

Required fields:
- `moneybird.token`: Your Moneybird personal token
- `moneybird.admin_id`: Your Moneybird administration ID
- `gmail.credentials_file`: Path to your Gmail OAuth credentials file
- `openai.api_key`: Your OpenAI API key


## Running

Build and run:

```bash
go build -o birdgpt cmd/main.go
./birdgpt
```

On first run:
1. You'll be prompted to visit a URL for Gmail authorization
2. After authorizing, copy the code and paste it back in the terminal
3. The application will start monitoring your emails

## License

BirdGPT is released under the MIT License. See the [LICENSE](LICENSE) file for more details.

## Security

If you discover any security-related issues, please email [me@janyk.dev](mailto:me@janyk.dev) instead of using the
issue tracker. All security vulnerabilities will be promptly addressed.

## Disclaimer

This project is not affiliated with Moneybird, OpenAI or Google. 