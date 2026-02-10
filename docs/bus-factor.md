# Bus Factor Analysis

The bus-factor analyzer measures code ownership concentration in a repository. A high bus factor indicates that knowledge is well-distributed; a low bus factor indicates risk from single-person dependencies.

## Concepts

### What is Bus Factor?

Bus factor is the minimum number of contributors whose absence would significantly impact a project's ability to continue development. Losing even one person on a low bus factor project can be catastrophic.

### Risk Levels

- **Critical**: 1-2 contributors account for most knowledge. Immediate action required.
- **High**: Less than 5 active contributors or significant concentration. Needs attention.
- **Medium**: Healthy but improvable. Some areas have limited coverage.
- **Low**: Knowledge well-distributed across 6+ contributors. Sustainable.

## Metrics

### Bus Factor

The minimum number of contributors needed to reach 50% of all commits in the analyzed scope. Lower values indicate higher risk.

```
Contributors: A(50%), B(25%), C(15%), D(10%)
Bus Factor: 1 (A alone = 50%)
```

### Herfindahl-Hirschman Index (HHI)

Measures market concentration; adapted for code ownership.

- **0.0**: Perfect diversity (equal contribution from all)
- **1.0**: Perfect concentration (one person has all commits)

Formula: HHI = Σ(market_share²), normalized to [0, 1]

### Gini Coefficient

Measures inequality in distribution of contributions.

- **0.0**: Perfect equality (everyone contributes equally)
- **1.0**: Perfect inequality (one person contributes everything)

### Top N %

Cumulative ownership percentage of the top N contributors:
- **Top 1 %**: Percentage from the single largest contributor
- **Top 3 %**: Percentage from the top 3 contributors
- **Top 5 %**: Percentage from the top 5 contributors

## Usage

### Basic Analysis

Analyze the current repository:

```bash
nightshift busfactor
```

Analyze a specific repository:

```bash
nightshift busfactor /path/to/repo
```

### Filtering

Analyze commits by specific file pattern:

```bash
nightshift busfactor --file "src/*.go"
```

Analyze commits within a date range:

```bash
nightshift busfactor --since 2024-01-01 --until 2024-12-31
```

### Output Formats

Human-readable report (default):

```bash
nightshift busfactor
```

JSON output for integration:

```bash
nightshift busfactor --json
```

### Saving Results

Save analysis results to the database:

```bash
nightshift busfactor --save
nightshift busfactor --save --db /path/to/nightshift.db
```

## Interpreting Results

### Example 1: Critical Risk

```
Risk Level: critical
Bus Factor: 1
Total Contributors: 1
Top 1 Contributor: 100.0%
Herfindahl Index: 1.0
```

**Interpretation**: Single-person project. Any absence is devastating.

**Actions**:
- Immediate knowledge transfer plan
- Code reviews from external team members
- Documentation of architecture and decisions
- Mentoring new contributors

### Example 2: High Risk

```
Risk Level: high
Bus Factor: 2
Total Contributors: 3
Top 1 Contributor: 60.0%
Top 3 Contributor: 95.0%
Herfindahl Index: 0.45
```

**Interpretation**: Project depends on 2 core people. Losing either is risky.

**Actions**:
- Pair programming sessions between experienced and new developers
- Code review policies requiring multiple approvals
- Architecture documentation and design records
- Gradual increase of contributor diversity

### Example 3: Low Risk

```
Risk Level: low
Bus Factor: 3
Total Contributors: 8
Top 1 Contributor: 20.0%
Top 3 Contributor: 55.0%
Herfindahl Index: 0.08
```

**Interpretation**: Well-distributed knowledge across team. Sustainable.

**Actions**:
- Maintain current practices
- Continue encouraging diverse contributions
- Ensure knowledge stays fresh through rotation

## CLI Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--json` | | Output as JSON |
| `--path` | `-p` | Repository or directory path |
| `--since` | | Start date (RFC3339 or YYYY-MM-DD) |
| `--until` | | End date (RFC3339 or YYYY-MM-DD) |
| `--file` | `-f` | Analyze specific file or pattern |
| `--save` | | Save results to database |
| `--db` | | Database path |

## Examples

### Analyze specific module

```bash
nightshift busfactor --file "internal/database/*"
```

### Recent changes

```bash
nightshift busfactor --since 2025-01-01
```

### Export results

```bash
nightshift busfactor --json > analysis.json
```

### Track changes over time

```bash
# Run regularly with --save
nightshift busfactor --save

# View historical trends
# Results stored in database at ~/.local/share/nightshift/nightshift.db
```

## Integration with Nightshift

Bus-factor analysis can be automated as a scheduled task:

```yaml
# nightshift.yaml
tasks:
  - name: "Bus Factor Check"
    type: analysis
    schedule: "0 9 * * MON" # Weekly Monday morning
    config:
      analyzer: bus-factor
      paths:
        - "."
      save-results: true
      alert-on-critical: true
```

## Metrics Accuracy

The analyzer uses git history, which means:

- **Accurate for**: Overall project health, identifying concentration
- **Not accurate for**: Real contribution value (one person might contribute more value in fewer commits)
- **Consider pairing with**: Code review patterns, architecture knowledge assessments

## Database Schema

Results are stored in the `bus_factor_results` table:

```sql
CREATE TABLE bus_factor_results (
    id              INTEGER PRIMARY KEY,
    component       TEXT NOT NULL,
    timestamp       DATETIME NOT NULL,
    metrics         TEXT,           -- JSON
    contributors    TEXT,           -- JSON
    risk_level      TEXT,
    report_path     TEXT
);
```

## Recommendations

### For Critical/High Risk Projects

1. **Documentation**: Create comprehensive architecture documentation
2. **Knowledge Transfer**: Regular pairing sessions between seniors and juniors
3. **Code Review**: Require reviews from multiple people
4. **Mentoring**: Formal mentorship programs
5. **Succession Planning**: Identify and develop successor contributors

### General Best Practices

- Track bus factor regularly (monthly or quarterly)
- Use as input for hiring decisions
- Include in project health assessments
- Set organizational targets (e.g., "bus factor > 3")
- Monitor trends over time
