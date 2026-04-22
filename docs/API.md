# API Reference

Ubuntu Excuses Data exposes a read-only JSON API over HTTP. The dataset is
loaded once at startup from `update_excuses.yaml` and served immutably. All
responses use `Content-Type: application/json` and support transparent gzip
compression via `Accept-Encoding: gzip`.

The server listens on `:8080` by default. Set the `ADDR` environment variable
to override (e.g. `ADDR=:9090`).

---

## Endpoints

### `GET /meta`

Returns dataset metadata and the distinct values available for filtering.

#### Response

| Field              | Type       | Description                                      |
|--------------------|------------|--------------------------------------------------|
| `generated_date`   | `string`   | ISO 8601 UTC timestamp of the dataset generation |
| `total_sources`    | `integer`  | Total number of source packages                  |
| `total_candidates` | `integer`  | Number of packages that are migration candidates |
| `components`       | `string[]` | Available component values for filtering         |
| `verdicts`         | `string[]` | Available verdict values for filtering           |
| `maintainers`      | `string[]` | Available maintainer values for filtering        |
| `arches`           | `string[]` | Architecture names present in test results       |
| `statuses`         | `string[]` | Autopkgtest status values (e.g. `PASS`, `REGRESSION`) |

#### Example

```
GET /meta
```

```json
{
  "generated_date": "2026-04-21T12:00:00Z",
  "total_sources": 4200,
  "total_candidates": 1500,
  "components": ["main", "universe", "restricted", "multiverse"],
  "verdicts": ["PASS", "REJECTED_PERMANENTLY", "REJECTED_TEMPORARILY"],
  "maintainers": ["Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>"],
  "arches": ["amd64", "arm64", "armhf", "ppc64el", "riscv64", "s390x"],
  "statuses": ["PASS", "REGRESSION", "RUNNING", "NEUTRAL"]
}
```

---

### `GET /sources`

Returns a paginated, optionally filtered and sorted list of source packages.

#### Query Parameters

| Parameter   | Type      | Default | Description                                                |
|-------------|-----------|---------|------------------------------------------------------------|
| `offset`    | `integer` | `0`     | Number of items to skip                                    |
| `limit`     | `integer` | `50`    | Maximum items to return (max `200`)                        |
| `sort`      | `string`  | `age`   | Sort field: `name` or `age`                                |
| `order`     | `string`  | `asc`   | Sort direction: `asc` or `desc`                            |
| `component` | `string`  | —       | Filter by component (e.g. `main`, `universe`)              |
| `verdict`   | `string`  | —       | Filter by migration policy verdict                         |
| `maintainer`| `string`  | —       | Filter by maintainer                                       |
| `status`    | `string`  | —       | Filter by migration status: `BLOCKED`, `WILL_ATTEMPT`, `WAITING`, `UNKNOWN` |

Multiple filters may be combined; results must match **all** specified filters.

#### Response

| Field            | Type       | Description                             |
|------------------|------------|-----------------------------------------|
| `generated_date` | `string`   | ISO 8601 UTC timestamp of the dataset   |
| `total`          | `integer`  | Total items matching the filters        |
| `offset`         | `integer`  | Current offset                          |
| `limit`          | `integer`  | Current limit                           |
| `sort`           | `string`   | Sort field used                         |
| `order`          | `string`   | Sort direction used                     |
| `sources`        | `Source[]`  | Array of source objects (see below)     |

#### Example

```
GET /sources?component=main&sort=age&order=desc&limit=2
```

```json
{
  "generated_date": "2026-04-21T12:00:00Z",
  "total": 850,
  "offset": 0,
  "limit": 2,
  "sort": "age",
  "order": "desc",
  "sources": [
    {
      "source_package": "example-pkg",
      "component": "main",
      "maintainer": "Ubuntu Developers <ubuntu-devel-discuss@lists.ubuntu.com>",
      "verdict": "PASS",
      "migration_status": "WILL_ATTEMPT",
      "old_version": "1.0-1",
      "new_version": "1.1-1",
      "is_candidate": true,
      "invalidated_by_other": false,
      "item_name": "example-pkg/1.1-1",
      "excuse": {
        "status": "WILL_ATTEMPT",
        "detail": "",
        "info": ["Maintainer: Ubuntu Developers"]
      },
      "policy_info": { "..." : "..." }
    }
  ]
}
```

---

### `GET /sources/{name}`

Returns the full details for a single source package.

#### Path Parameters

| Parameter | Type     | Description              |
|-----------|----------|--------------------------|
| `name`    | `string` | Source package name       |

#### Response

Returns a single [Source object](#source-object).

#### Errors

| Status | Body                                        | Condition           |
|--------|---------------------------------------------|---------------------|
| `404`  | `{"error": "source not found: <name>"}`     | Package not found   |

#### Example

```
GET /sources/systemd
```

---

### `GET /sources/{name}/autopkgtest`

Returns the autopkgtest policy results for a single source package.

#### Path Parameters

| Parameter | Type     | Description              |
|-----------|----------|--------------------------|
| `name`    | `string` | Source package name       |

#### Response

| Field      | Type                              | Description                                    |
|------------|-----------------------------------|------------------------------------------------|
| `verdict`  | `string`                          | Overall autopkgtest verdict                    |
| `packages` | `map[string]map[string]Result`    | Per-package, per-architecture test results      |

Each `Result` contains:

| Field     | Type      | Description                               |
|-----------|-----------|-------------------------------------------|
| `status`  | `string`  | Test status (e.g. `PASS`, `REGRESSION`)   |
| `log_url` | `string`  | Optional URL to the test log; omitted when unavailable |
| `pkg_url` | `string`  | Optional URL to the package info; omitted when unavailable |

#### Errors

| Status | Body                                        | Condition           |
|--------|---------------------------------------------|---------------------|
| `404`  | `{"error": "source not found: <name>"}`     | Package not found   |

#### Example

```
GET /sources/systemd/autopkgtest
```

```json
{
  "verdict": "PASS",
  "packages": {
    "systemd/257.4-1ubuntu3": {
      "amd64": { "status": "PASS", "log_url": "https://...", "pkg_url": "https://..." },
      "arm64": { "status": "PASS", "log_url": "https://...", "pkg_url": "https://..." }
    }
  }
}
```

---

## Source Object

The full source object returned by `/sources` and `/sources/{name}`:

| Field                  | Type                | Description                                          |
|------------------------|---------------------|------------------------------------------------------|
| `source_package`       | `string`            | Package name                                         |
| `component`            | `string`            | Archive component (e.g. `main`)                      |
| `maintainer`           | `string`            | Package maintainer                                   |
| `verdict`              | `string`            | Migration policy verdict                             |
| `migration_status`     | `string`            | High-level status: `BLOCKED`, `WILL_ATTEMPT`, `WAITING`, `UNKNOWN` |
| `old_version`          | `string`            | Current version in the target suite                  |
| `new_version`          | `string`            | Proposed version in the source suite                 |
| `is_candidate`         | `boolean`           | Whether the package is a migration candidate         |
| `invalidated_by_other` | `boolean`           | Whether another package invalidated this entry       |
| `item_name`            | `string`            | Item identifier (usually `name/version`)             |
| `excuse`               | `Excuse`            | Migration excuse details                             |
| `policy_info`          | `PolicyInfo`        | Per-policy verdicts and details                      |
| `dependencies`         | `Dependencies?`     | Dependency information (omitted if none)             |
| `hints`                | `Hint[]?`           | Migration hints (omitted if none)                    |
| `reason`               | `string[]?`         | Reason strings (omitted if none)                     |

### Excuse

| Field    | Type       | Description                                             |
|----------|------------|---------------------------------------------------------|
| `status` | `string`   | Migration status                                        |
| `detail` | `string`   | Detail text after the status (omitted when empty)       |
| `info`   | `string[]` | Additional informational lines (omitted when empty)     |

### PolicyInfo

| Field           | Type                    | Description                              |
|-----------------|-------------------------|------------------------------------------|
| `age`           | `AgePolicy`             | Age policy details                       |
| `autopkgtest`   | `AutopkgtestPolicy`     | Autopkgtest results                      |
| `block`         | `string`                | Block policy verdict                     |
| `block_bugs`    | `string`                | Block-bugs policy verdict                |
| `depends`       | `string`                | Depends policy verdict                   |
| `email`         | `string`                | Email policy verdict                     |
| `linux`         | `string?`               | Linux policy verdict (omitted if n/a)    |
| `rc_bugs`       | `RcBugsPolicy`          | RC bugs policy details                   |
| `source_ppa`    | `string`                | Source PPA policy verdict                |
| `update_excuse` | `UpdateExcusePolicy`    | Update excuse policy details             |

### AgePolicy

| Field             | Type      | Description                       |
|-------------------|-----------|-----------------------------------|
| `age_requirement` | `integer` | Required age in days              |
| `current_age`     | `number`  | Current age in days               |
| `verdict`         | `string`  | Age policy verdict                |

### RcBugsPolicy

| Field               | Type                | Description                           |
|---------------------|---------------------|---------------------------------------|
| `shared_bugs`       | `integer[]`         | Bug IDs shared between source/target  |
| `unique_source_bugs`| `integer[]`         | Bug IDs unique to source              |
| `unique_target_bugs`| `integer[]`         | Bug IDs unique to target              |
| `verdict`           | `string`            | RC bugs verdict                       |

### UpdateExcusePolicy

| Field     | Type                     | Description                                          |
|-----------|--------------------------|------------------------------------------------------|
| `verdict` | `string`                 | Update excuse verdict                                |
| `bugs`    | `object<string, integer>`| Launchpad bug ID → last-updated Unix timestamp       |

### Dependencies

| Field           | Type       | Description                                   |
|-----------------|------------|-----------------------------------------------|
| `blocked_by`    | `string[]` | Packages blocking this source                 |
| `migrate_after` | `string[]` | Packages this source must migrate after       |

### Hint

| Field  | Type     | Description                  |
|--------|----------|------------------------------|
| `from` | `string` | Who issued the hint          |
| `type` | `string` | Hint type (e.g. `unblock`)   |

---

## Error Responses

All error responses have the form:

```json
{
  "error": "descriptive message"
}
```

| Status | Meaning                          |
|--------|----------------------------------|
| `404`  | Resource not found               |
| `500`  | Internal server error            |
