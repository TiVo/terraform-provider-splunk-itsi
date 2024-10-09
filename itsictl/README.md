# itsictl

`itsictl` is a command-line tool for performing common operations with Splunk IT Service Intelligence (ITSI). Currently it allows you to perform various operations mostly related to managing KPI thresholds, in particular resetting thresholds or applying ML-assisted thresholds.

## Table of Contents

- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
  - [General Options](#general-options)
  - [Authentication Options](#authentication-options)
  - [Commands](#commands)
    - [threshold](#threshold-command)
      - [reset](#reset-command)
      - [recommend](#recommend-command)
- [Examples](#examples)

---

## Features

- Reset thresholds for specified KPIs/services.
- Apply machine learning-assisted thresholds based on historical KPI data.
- Flexible targeting of services and KPIs using selectors with wildcard support.
- Dry run mode to preview changes without applying them.

## Installation

You have two primary options to install `itsictl`:

- **Build from Source Using Homebrew Formula (recommended)**
- **Build from Source Manually**

### Build from Source Using Homebrew Formula


**Prerequisites:**

- Homebrew installed on your system. If you don't have Homebrew, you can install it by following the instructions at [brew.sh](https://brew.sh/).

**Installation Steps:**

1. **Install `itsictl` Using Homebrew:**

  Run the following command to install `itsictl` from the latest source:

  ```bash
    git clone git@github.com:TiVo/terraform-provider-splunk-itsi.git
    cd terraform-provider-splunk-itsi
    brew install --HEAD --build-from-source ./itsictl/brew/itsictl.rb
  ```

   This command will:

   - Build `itsictl` from the latest source code.
   - Install the binary into your system.
   - Install manpages and shell completion scripts.

**Note:** After installation or updating, you may need to refresh your shell's completion cache to enable autocompletion.

2. **Verify Installation:**

  ```bash
  itsictl version
  ```

**Updating `itsictl`:**

If `itsictl` is already installed and you want to update it to the latest version, run:

```bash
    cd terraform-provider-splunk-itsi
    brew reinstall --HEAD --build-from-source ./itsictl/brew/itsictl.rb
```

This will uninstall the existing version and install the latest version from the source.

### Build from Source Manually

Ensure you have Go installed (version 1.23 or later).

```bash
git clone git@github.com:TiVo/terraform-provider-splunk-itsi.git
cd terraform-provider-splunk-itsi
make build
```


## Configuration

`itsictl` can be configured using a configuration file, environment variables, or command-line flags.

### Configuration File

By default, `itsictl` looks for a configuration file named `.itsictl.yaml` in your home directory. You can specify a different config file using the `--config` flag.

Example `.itsictl.yaml`:

```yaml
host: itsi.example.com
port: 8089
user: admin
password: yourpassword
insecure: false
concurrency: 10
verbose: true
```

### Environment Variables

You can set environment variables prefixed with `ITSICTL_` to configure the tool.

Example:

```bash
export ITSICTL_HOST=itsi.example.com
export ITSICTL_USER=admin
export ITSICTL_PASSWORD=yourpassword
```

### Command-Line Flags

You can override configuration options using command-line flags.

Example:

```bash
itsictl --host itsi.example.com --user admin --password yourpassword threshold reset --service service1
```

## Usage

```bash
itsictl [command] [flags]
```

### General Options

- `--config`: Specify the config file (default is `$HOME/.itsictl.yaml`).
- `-v`, `--verbose`: Enable verbose output.
- `--concurrency`: Number of concurrent operations (default is 10).

### Authentication Options

- `--host`: Splunk ITSI host (default is `localhost`).
- `--port`: Splunk ITSI port (default is `8089`).
- `--insecure`: Disable TLS certificate verification.
- `--access-token`: Access token for authentication.
- `--user`: Username for authentication (default is `admin`).
- `--password`: Password for authentication.

### Commands

#### `threshold` Command

Manage KPI thresholds.

```bash
itsictl threshold [subcommand] [flags]
```

**Subcommands:**

- `reset`: Reset thresholds for specified KPIs/services.
- `recommend`: Apply machine learning-assisted thresholds to specified KPIs/services.

**Common Flags for `threshold` Subcommands:**

- `-s`, `--service`: Specify one or more Service IDs or names (can be used multiple times).
- `-k`, `--kpi`: Specify one or more KPI IDs or names (can be used multiple times).
- `--dry-run`: Perform a dry run without making any changes.

##### `reset` Command

Reset the threshold configurations for specified KPIs or services to their default state.

**Usage:**

```bash
itsictl threshold reset [flags]
```

**Description:**

- Resets the following threshold settings for the matching KPIs:
  - Adaptive thresholds: Disabled
  - Aggregate thresholds: Set to 'Normal' severity
  - Entity thresholds: Set to 'Normal' severity
  - Time variate thresholds: Disabled
  - Outlier detection: Disabled

**Important:**

- To prevent accidental resetting of thresholds for all services, the `reset` command requires at least one service or KPI selector to be provided using the `--service` or `--kpi` flags.

##### `recommend` Command

Apply machine learning-assisted thresholds to specified KPIs or services.

**Usage:**

```bash
itsictl threshold recommend [flags]
```

**Description:**

- Performs historical data analysis on matching KPIs configured to use ML-assisted thresholds.
- Updates threshold configurations based on ML recommendations.

**Flags:**

- `--use-latest-data`: Use the latest available KPI data for analysis, ignoring the stored starting date.
- `--insufficient-data-action`: Action to take for KPIs with insufficient data (`skip` or `reset`, default is `skip`).

**Notes:**

- KPIs using threshold templates or custom thresholds are skipped.
- If the ML analysis cannot recommend thresholds due to insufficient data or constant values, the default behavior is to skip the KPI and retain its current configuration.
- Use `--insufficient-data-action reset` to reset the threshold configuration in such cases.

## Examples

### Reset Thresholds

- **Reset thresholds for all KPIs in a specific service:**

  ```bash
  itsictl threshold reset --service service1
  ```

- **Reset thresholds for all KPIs in multiple services:**

  ```bash
  itsictl threshold reset --service service1 --service service2
  ```

- **Reset thresholds for specific KPIs in services matching a pattern:**

  ```bash
  itsictl threshold reset --service "sample service*" --kpi errors --kpi "network*"
  ```

- **Perform a dry run (no changes will be made):**

  ```bash
  itsictl threshold reset --service service1 --dry-run
  ```

### Apply ML-Assisted Thresholds

- **Apply ML-assisted thresholds to all KPIs in a specific service:**

  ```bash
  itsictl threshold recommend --service service1
  ```

- **Apply ML-assisted thresholds to all KPIs in multiple services:**

  ```bash
  itsictl threshold recommend --service service1 --service service2
  ```

- **Apply ML-assisted thresholds to specific KPIs in services matching a pattern:**

  ```bash
  itsictl threshold recommend --service "sample service*" --kpi errors --kpi "network*"
  ```

- **Use the latest data for analysis, ignoring the stored starting date:**

  ```bash
  itsictl threshold recommend --service service1 --use-latest-data
  ```

- **Reset threshold configurations for KPIs with insufficient data:**

  ```bash
  itsictl threshold recommend --service service1 --insufficient-data-action reset
  ```

- **Perform a dry run to preview changes:**

  ```bash
  itsictl threshold recommend --service service1 --dry-run
  ```

---

**Note:** Always review the commands and flags carefully to ensure you're targeting the correct services and KPIs. Use the `--dry-run` flag to preview actions before applying changes.

For further help on a specific command, use:

```bash
itsictl [command] --help
```

Example:

```bash
itsictl threshold recommend --help
```

