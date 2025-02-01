# Toonami Aftermath CLI

This is a CLI for the Toonami Aftermath website. It allows you to scrape the M3U Playlist and XMLTV Guide from the website so that you can import it into your own IPTV player.

## Features
* Scrape the M3U Playlist and XMLTV Guide from the Toonami Aftermath website
* Can run on a cron schedule within the binary (no need for external cron)

## Getting Started

### Installation

#### Binary
Download the appropriate binary for your OS/CPU from releases.

#### Docker
```shell
# Your local directory in this case contains your `config.yaml` file.
docker run -v $(pwd):/config ghcr.io/chris102994/toonamiaftermath-cli:latest
```

### Usage


#### Run
```bash
# Help
./toonamiaftermath-cli run --help
Run the toonamiaftermath-cli

Usage:
  toonamiaftermath run [flags]

Flags:
  -c, --cron-expression string   The cron schedule to run the command
  -h, --help                     help for run
  -m, --m3u-output string        Path to the M3U output file (default "index.m3u")
  -x, --xmltv-output string      Path to the XMLTV output file (default "index.xml")

Global Flags:
      --config string       Path to the configuration file
  -f, --log-format string   Log format (text, json) (default "text")
  -l, --log-level string    Log level (trace, debug, info, warn, error, fatal, panic) (default "info")
```

```bash
# Run Once
./toonamiaftermath-cli run
...
```

```bash
# Run on a schedule
./toonamiaftermath-cli run --cron-expression "@every 12h"
```

### Configuration Variables, Files and flags.

| Env Variable      | Config File Variable | Description                                                                |
|-------------------|----------------------|----------------------------------------------------------------------------|
| `LOGGING_LEVEL`   | `logging.level`      | Log level (trace, debug, info, warn, error, fatal, panic) (default "info") |
| `LOGGING_FORMAT`  | `logging.format`     | Log format (text, json) (default "text")                                   |
| `CRON_EXPRESSION` | `cron.expression`    | The cron schedule to run the command                                       |
| `M3U_OUTPUT`      | `run.m3u_output`     | Path to the M3U output file (default "index.m3u")                          |
| `XMLTV_OUTPUT`    | `run.xmltv_output`   | Path to the XMLTV output file (default "index.xml")                        |

##### Example Configuration File

Configuration file should support any format supported by [spf13/viper](https://github.com/spf13/viper).

```yaml
---
logging:
  level: "trace"
  format: "text"
cron:
  expression: "@every 12h"
run:
  xmltv_output: "index.xml"
  m3u_output: "index.m3u"
```