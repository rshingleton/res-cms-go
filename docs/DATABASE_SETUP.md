# Database Setup

ResCMS Go supports multiple database backends via GORM. You can configure your preferred database in `rescms.yml`.

## 📦 Supported Databases

- **SQLite** (Default, recommended for small to medium sites)
- **MySQL** / **MariaDB**
- **PostgreSQL**

---

## 1. SQLite (Default)

SQLite is the simplest to set up as it requires no external server. The database is stored in a single file on disk.

### Configuration
In `rescms.yml`:

```yaml
database:
  type: sqlite
  path: data/rescms.db
  wal_mode: true # Recommended for better concurrency
```

- **path**: The relative or absolute path to the `.db` file.
- **wal_mode**: Enables Write-Ahead Logging for improved performance.

---

## 2. MySQL / MariaDB

MySQL and MariaDB are excellent choices for larger sites or when using external database hosting.

### Prerequisites
1. Install MySQL or MariaDB server.
2. Perform administrative setup (requires **root/sudo** access to the database server):
   ```sql
   /* Connect as root: mysql -u root -p */
   CREATE DATABASE IF NOT EXISTS rescms CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
   CREATE USER 'rescms_user'@'localhost' IDENTIFIED BY 'your_password';
   GRANT ALL PRIVILEGES ON rescms.* TO 'rescms_user'@'localhost';
   FLUSH PRIVILEGES;
   ```

### Configuration
In `rescms.yml`:

```yaml
database:
  type: mysql
  dsn: "rescms_user:your_password@tcp(127.0.0.1:3306)/rescms?charset=utf8mb4&parseTime=True&loc=Local"
```

- **dsn**: Data Source Name format: `username:password@tcp(host:port)/dbname?params`.

---

## 3. PostgreSQL

PostgreSQL is a robust, open-source relational database.

### Prerequisites
1. Install PostgreSQL server.
2. Create a new user and database (requires **sudo** to switch to the `postgres` system user):
   ```bash
   # Login as postgres system user
   sudo -u postgres psql

   # Inside the psql prompt:
      CREATE DATABASE rescms;
   CREATE USER rescms_user WITH ENCRYPTED PASSWORD 'your_password';
   GRANT ALL PRIVILEGES ON DATABASE rescms TO rescms_user;

   -- Required for PostgreSQL 15+: grant schema permissions
   \c rescms
   GRANT ALL ON SCHEMA public TO rescms_user;

   -- Optional: allow the user to create databases (for automatic bootstrapping)
   ALTER USER rescms_user WITH CREATEDB;

   ```

### Configuration
In `rescms.yml`:

```yaml
database:
  type: postgres
  dsn: "host=localhost user=rescms_user password=your_password dbname=rescms port=5432 sslmode=disable TimeZone=UTC"
```

- **dsn**: Connection string format: `host=... user=... password=... dbname=... port=... sslmode=...`.

---

## 🔄 Automatic Bootstrapping

ResCMS Go attempts to automatically create the database if it doesn't exist, provided the user in your DSN has sufficient permissions (e.g., `CREATEDB` in Postgres).

If the automatic bootstrapping fails with a "database does not exist" error, ensure you have followed the **Prerequisites** section above to create the database manually using `sudo`.

Once configured, run the application:
```bash
go run cmd/res-cms/main.go
```
