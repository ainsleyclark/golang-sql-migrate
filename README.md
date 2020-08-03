# golang-sql-migrate

__A simple yet versatile mysql migrate library package for golang__

golang-sql-migrate is extremely lightweight and efficient. has only 1 dependency for verbose logging which can be turned off in the migrate constructor. The following commands can utilised:
* Up
* Down
* Rollback
* Fresh
* Drop
* Drop and Create
* Make


## Install
`` go get github.com/ainsleyclark/golang-sql-migrate``

## Getting Started
To use simply create a new migration instance and pass in the following:

| Parameter | Type | Example | Description |
|-----------|------|---------|-------------|
| `database` | `*sql.DB` | `-` | Database of type database/sql. |
| `databaseName` | `string` | `mydatabase` | The database name. |
| `migrationPath` | `string` | `/Users/me/code/migrations` | The path where the migrations are stored. |
| `verbose` | `bool` | `true` | To enable logging in shell. |

__Example__
```
m, err := migrate.NewInstance(
    db.DB,
    "go_cms",
    "/Users/ainsley/Desktop/Reddico/apis/cms/api/database/migrations/",
    false,
)
```

This command returns a new instance of migrate, which you are then able to call the functions on below.

## Functions

## Suggestions
Please feel free to make any suggestions to the package.

## Contributing
We welcome contributors, but please read the contributing document before making a pull request. You can also browse the issues or help wanted section found above.

