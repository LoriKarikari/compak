# Nextcloud Example Package

This is a complete Nextcloud installation with PostgreSQL database and Redis caching.

## Quick Start

```bash
# Install Nextcloud
compak install ./examples/nextcloud \
  --set admin_password=your_admin_password \
  --set db_password=your_db_password \
  --set redis_password=your_redis_password

# Access Nextcloud
# Open http://localhost:8080 in your browser
```

## Configuration

The following parameters can be customized:

- `port` - Port to expose Nextcloud (default: 8080)
- `admin_user` - Admin username (default: admin)
- `admin_password` - Admin password (required)
- `db_password` - PostgreSQL password (required)
- `redis_password` - Redis password (required)

## Custom Configuration Example

```bash
compak install ./examples/nextcloud \
  --set port=9000 \
  --set admin_user=myadmin \
  --set admin_password=secure_admin_pass \
  --set db_password=secure_db_pass \
  --set redis_password=secure_redis_pass
```
