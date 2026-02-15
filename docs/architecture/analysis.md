# Analysis & Bus-Factor Metrics

Nightshift uses advanced static analysis of Git history to identify technical debt and maintenance risks. The primary focus is on "Bus-Factor" — the minimum number of contributors whose loss would jeopardize the project.

## Data Extraction
The `internal/analysis` package uses the local Git binary to extract commit history.
- **GitParser**: Executes `git log` with specific formatting (`%an|%ae`) to reliably identify authors by name and email.
- **Filtering**: Supports date ranges (e.g., "last 6 months") and file-path filtering to analyze specific components or sub-directories.

## Ownership Metrics
Once data is extracted, Nightshift calculates several industry-standard metrics to quantify risk:

### 1. Bus-Factor
The minimum number of top contributors required to account for 50% of the total commits in the analyzed period. A low bus-factor (e.g., 1 or 2) indicates high risk.

### 2. Herfindahl-Hirschman Index (HHI)
A measure of market concentration, adapted for code ownership. 
- **Scale**: 0 (perfectly diverse) to 1 (monopoly).
- **Formula**: Sum of the squares of each contributor's "market share" of commits.
- **Interpretation**: High HHI suggests knowledge is concentrated in a few individuals.

### 3. Gini Coefficient
A measure of inequality typically used for income, adapted for commit distribution.
- **Scale**: 0 (all contributors have equal commits) to 1 (one contributor did everything).

### 4. Top N Concentration
Calculates the percentage of ownership held by the top 1, 3, and 5 contributors.

## Risk Assessment Logic
Nightshift categorizes risk into four levels based on the calculated metrics:

| Risk Level | Primary Criteria |
|:---|:---|
| **Critical** | Top 1 contributor > 80% OR total contributors ≤ 1 |
| **High** | Top 1 contributor > 50% OR top 2 contributors > 80% OR total contributors ≤ 2 |
| **Medium** | HHI > 0.3 OR total contributors ≤ 5 |
| **Low** | All other cases (well-distributed knowledge) |

## Maintenance Recommendations
Based on the risk level, the analysis engine generates actionable recommendations:
- **Knowledge Transfer**: Suggesting pairing sessions or documentation tasks.
- **Code Reviews**: Identifying specific areas that require reviews from non-primary owners.
- **Junior Engagement**: Encouraging contributions to components with high concentration to diversify ownership.
