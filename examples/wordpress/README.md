# WordPress Example Package

This is a complete WordPress installation with MariaDB database.

## Quick Start

```bash
# Install WordPress
compak install ./examples/wordpress --set db_password=your_secure_password

# Access WordPress
# Open http://localhost:8080 in your browser
```

## Configuration

The following parameters can be customized:

- `port` - Port to expose WordPress (default: 8080)
- `db_password` - Database password (required)
- `db_name` - Database name (default: wordpress)
- `db_user` - Database user (default: wordpress)

## Custom Port Example

```bash
compak install ./examples/wordpress \
  --set port=9000 \
  --set db_password=your_secure_password
```
