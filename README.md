# Bitbucket Commit Fetcher

This is a Go script that fetches commits from Bitbucket repositories and writes them into a JSON file. The commits are fetched from all repositories under Bitbucket projects. The fetched commit data includes information such as the commit hash, author, date, message, and links to the commit and patch.

## Table of Contents

- [Running cli](#running-cli)
- [How It Works](#how-it-works)
- [Contributing](#contributing)
- [License](#license)


## Running cli

To run the script:

1. Clone the repository:

```bash
git clone https://github.com/nyxragon/bitybuckety.git
```

2. Navigate to the `cli` directory:

```bash
cd cli
```

3. Run the script:

```bash
go run main.go
```

By default, the commits are 100 and the date is `2024-12-20T00:00:00+00:00`.

To specify custom parameters:

- **-date**  
  Fetch commits before this date (default `2024-12-20T00:00:00+00:00`).
  
- **-total**  
  Total number of commits to fetch (default `100`).

### Example Usage

```bash
go run main.go -date "2024-12-15T00:00:00+00:00" -total 50
```

This command will fetch 50 commits before `2024-12-15T00:00:00+00:00`.

## How It Works

This script fetches commits from all Bitbucket repositories in a given account or project. It fetches commits in batches, processes the data, and writes it into a JSON file. The file is named based on the current timestamp in the format `commits_YYYY-MM-DD.json`.

The script utilizes the Bitbucket API to fetch the necessary data.

## Contributing

Contributions are welcome, and future enhancements are encouraged.

## License

This project is licensed under the GPU License - see the [LICENSE](LICENSE) file for details.