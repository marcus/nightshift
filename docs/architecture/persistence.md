# Persistence & State Management

Nightshift uses a local persistence layer to track run history, manage task cooldowns, and ensure consistent behavior across restarts.

## Data Storage: SQLite
The primary storage engine is **SQLite**, implemented using the `modernc.org/sqlite` driver. This choice ensures:
- **Zero Dependencies**: A CGO-free implementation that simplifies cross-compilation and deployment.
- **Local-First**: Data stays on the user's machine, typically in `~/.local/share/nightshift/nightshift.db`.
- **Concurrency**: Uses Write-Ahead Logging (WAL) mode to support concurrent read/write operations safely.

## Key Data Entities

### 1. Projects
Tracks every repository Nightshift has encountered.
- **Path**: Canonical local path to the repository.
- **Last Run**: Timestamp of the last successful processing.
- **Run Count**: Total number of times the project has been analyzed.

### 2. Task History
Tracks individual task types per project to manage "cooldown" periods.
- **Project Path + Task Type**: Unique identifier for a task execution context.
- **Last Run**: Used to calculate if a task is eligible to run again based on its configured interval.

### 3. Run Records
A detailed audit trail of every Nightshift execution.
- **Run ID**: Unique UUID for the run.
- **Metrics**: Start/end times, tokens used, and status (success/failure).
- **Context**: Which provider was used and which projects/tasks were processed.

## Lifecycle & Management

Nightshift takes full responsibility for the lifecycle of the local database:

### 1. Automatic Instantiation
The database is automatically created during the first run if it does not exist.
- **Location**: Typically `~/.local/share/nightshift/nightshift.db`.
- **Directory Security**: Nightshift creates the storage directory with `0700` permissions (read/write/execute for the owner only) to ensure local data privacy.

### 2. Schema Management & Migrations
Nightshift uses a built-in migration system (found in `internal/db/migrations.go`) to manage the SQLite schema:
- **Auto-Update**: On every startup, Nightshift checks the current schema version against the compiled migrations and applies any missing updates within a transaction.
- **Version Tracking**: A `schema_version` table keeps track of the applied migrations.
- **No Manual Setup**: Users never need to manually run SQL scripts or initialize tables.

### 3. User & Password Management
Because Nightshift uses **SQLite**, which is a serverless, file-based database engine:
- **No Database Users**: There is no concept of database users, roles, or grants.
- **No Passwords**: Access control is handled entirely by **filesystem permissions**. Since the database file is stored in the user's home directory with restricted permissions, only the user (and the Nightshift process running as that user) can access the data.
- **Simplicity**: This removes the overhead of credential management for the local persistence layer.

## State Logic
Persistence is managed by the `internal/state` package, which provides the high-level API used by the Orchestrator:
- **Staleness Calculation**: Determines if a project or task needs attention by comparing `Last Run` with current configuration.
- **Run Prevention**: Ensures that a project isn't processed multiple times within a single 24-hour window unless forced.
- **Snapshotting**: (Via `internal/snapshots`) allows for rolling back or inspecting project state before/after maintenance.
