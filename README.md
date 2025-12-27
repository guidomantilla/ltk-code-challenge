# LTK Code Challenge - Events API

This project is a performant, idiomatic Golang RESTful API service for managing a collection of "Events", with PostgreSQL integration.

## Prerequisites

- [Go](https://golang.org/doc/install) (version 1.21+ recommended)
- [Docker](https://docs.docker.com/get-docker/) and [Docker Compose](https://docs.docker.com/compose/install/)
- [direnv](https://direnv.net/) (recommended for environment variable management)

### Install Tools
To install the necessary development tools (linters, coverage tools, etc.):
```bash
make install
```

## Configuration

This project uses `direnv` to automatically load environment variables from the `.envrc` file when you enter the project directory.

Make sure you have `direnv` installed and allow it for this project:

```bash
direnv allow
```

The following environment variables are required (already defined in `.envrc`):

- `DB_NAME`: Name of the PostgreSQL database.
- `DB_USER`: PostgreSQL user.
- `DB_PASSWORD`: PostgreSQL password.
- `DB_HOST`: Database host (default: `localhost`).
- `DB_PORT`: Database port (default: `5433`).

## Database Setup

To start the PostgreSQL database container, use the following command:

```bash
docker compose -f docker/docker-compose.yml -p ltk up --detach --remove-orphans
```

This will start a PostgreSQL instance with the credentials and database name defined in the `docker-compose.yml` file, which match the default values in `.envrc`.

## Running the Application

Once the database is running and environment variables are loaded, you can start the service:

```bash
go run main.go
```

The server will start on `http://localhost:8080`.

## API Documentation

### Create Event
- **URL**: `/events`
- **Method**: `POST`
- **Body**:
  ```json
  {
    "title": "Project Kickoff",
    "description": "Initial meeting for the new project",
    "start_time": "2025-12-23T10:00:00Z",
    "end_time": "2025-12-23T11:00:00Z"
  }
  ```
- **Success Response**: `201 Created` with the saved event object.

#### Example:
  ```
curl --location 'http://localhost:8080/events/' \
--header 'Content-Type: application/json' \
--data '{
"title": "An Event",
"start_time": "2025-12-22T10:00:00Z",
"end_time": "2025-12-22T11:00:00Z"
}'
  ```

### Get Event by ID
- **URL**: `/events/:id`
- **Method**: `GET`
- **Success Response**: `200 OK` with the event object or `404 Not Found` if it doesn't exist.

#### Example:
  ```
curl --location 'http://localhost:8080/events/4acc05c4-3526-4c09-8739-621c4b57c8e6'
  ```

## Development and Quality Checks

The project includes a `Makefile` to automate common development tasks and quality checks.





### Full Validation
You can run a full validation of the project using the following command:

```bash
make validate
```

The `make validate` command performs the following steps:
- **Imports**: Updates and formats Go imports (`make imports`).
- **Format**: Formats the code using `go fmt` (`make format`).
- **Vet**: Runs `go vet` to find common errors (`make vet`).
- **Lint**: Executes `golangci-lint` to ensure code quality (`make lint`).
- **Coverage**: Runs unit tests and generates coverage reports (`make coverage`).
- **Check**: Performs vulnerability checks using `govulncheck` (`make check`).

### Individual Targets
You can also run these steps individually:
- `make test`: Run unit tests.
- `make coverage`: Run tests and generate HTML coverage report in `.reports/testcoverage.html`.
- `make lint`: Run the linter.
- `make report`: Generate various quality reports in the `.reports/` directory.
