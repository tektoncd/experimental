# Results API Server

## Variables

| Environment Variable | Description                     | Example                                      |
| -------------------- | ------------------------------- | -------------------------------------------- |
| DB_USER              | MySQL Database user             | user                                         |
| DB_PASSWORD          | MySQL Database Password         | hunter2                                      |
| DB_PROTOCOL          | MySQL Database Network Protocol | unix                                         |
| DB_ADDR              | MySQL Database address          | /cloudsql/my-project:us-east1:tekton-results |
| DB_NAME              | MySQL Database name             | tekton_results                               |

Values derived from MySQL DSN (see
https://github.com/go-sql-driver/mysql#dsn-data-source-name)
