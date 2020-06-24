# Backend Service
Backend service provides REST APIs for the Pipelines-Marketplace UI to interact
with the database. It also povides file service by caching the YAML and README
files from the Github reposiory provided by the user.

### Dependencies
1. Go 1.13.3
2. PostgreSQL 10.6

### Running on your local machine
1. Fork and clone this repository
2. Create a .env file with the following fields <br/>
```
GITHUB_TOKEN=""
POSTGRESQL_USERNAME=""
POSTGRESQL_PASSWORD=""
POSTGRESQL_DATABASE=""
HOST=""
PORT=
CLIENT_ID=""
CLIENT_SECRET=""
```

Get your Github Access token from <https://github.com/settings/tokens>

3. Install dependencies
  ```go mod download```

4. Restore the latest database backup by executing the below command with HOST, PORT, DB_NAME AND USER_NAME
```pg_restore -h HOST -p PORT -d DB_NAME -U USER_NAME latest_database_backup.dump```

5. Build the application
```go build -o bin/ ./cmd/...```

6. Run
```./backend```
