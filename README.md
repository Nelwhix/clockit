# clockit

A time tracking CLI for freelancers working with multiple companies.

## Quickstart

### Install

Install the latest version with Go:

```sh
go install github.com/Nelwhix/clockit@latest
```

Or install from a local checkout:

```sh
git clone https://github.com/Nelwhix/clockit.git
cd clockit
go install .
```

Make sure your Go binary directory is on your `PATH`. For most Go setups, that is:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Initialize clockit

Create the local SQLite data store before using the plugin:

```sh
clockit init
```

If you need to reset the local data store, run:

```sh
clockit init --force
```

`--force` deletes existing clockit data, so use it carefully.

### Track Your First Task

Create a company:

```sh
clockit company create --name "Acme Inc" --rate-cents 7500 --currency USD
```

Create a task for that company:

```sh
clockit task create --name "Website redesign" --company "Acme Inc" --description "Client website work"
```

List tasks to find the task ID:

```sh
clockit task list
```

Start tracking time:

```sh
clockit task start 1
```

Press `q` or `Ctrl+C` to stop tracking and save the time entry.

### Generate a Report

Generate a task breakdown for a date range:

```sh
clockit report --from 2026-06-01 --to 2026-06-30
```

Filter by task ID:

```sh
clockit report --from 2026-06-01 --to 2026-06-30 --task-id 1
```

Filter by task name:

```sh
clockit report --from 2026-06-01 --to 2026-06-30 --task-name "Website redesign"
```

## Common Commands

```sh
clockit company list
clockit company delete --name "Acme Inc" --yes

clockit task list
clockit task list --all
clockit task update --id 1 --name "Website refresh"
clockit task delete --id 1 --yes
```

Use built-in help to see all available commands and flags:

```sh
clockit --help
clockit company --help
clockit task --help
clockit report --help
```

## Data Location

clockit stores its SQLite database in a platform-specific application data directory:

- macOS: `~/Library/Application Support/clockit/clockit.db`
- Linux: `$XDG_DATA_HOME/clockit/clockit.db` or `~/.local/state/clockit/clockit.db`
- Windows: `%LocalAppData%\clockit\clockit.db`

For tests or isolated runs, set `CLOCKIT_TEST_DB` to a custom database path.
