---
---
# Renovate {params.release} release train stats

In Renovate {params.release}.x, there were:

- <Value data={num_commits} /> commits
- <Value data={num_releases} /> releases
- <Value data={num_minor_releases} /> minor releases
- <Value data={longest_commit_streak} column=days /> consecutive days of commits

And had the following release cadence:

<BarChart data={all_release_dates} series="release_type" />

Given the conventional commit type, we also had:

<DataTable data={commits_by_type} />

And excluding dependency updates:

<DataTable data={commits_by_type_no_deps} />

Based on the conventional commit scope, the top 10 scopes were:

<DataTable data={top_10_scopes} />

Dates with most releases:

<DataTable data={top_release_dates} />

Longest stream of commits: <Value data={longest_commit_streak} column=days /> between <Value data={longest_commit_streak} column=start_date /> and <Value data={longest_commit_streak} column=end_date />

```sql num_commits
select
    COUNT(*) as total_commits
from
    commits
where
    major_version = ${params.release}
```


```sql num_commits
select
    COUNT(*) as total_commits
from
    commits
where
    major_version = ${params.release}
```

```sql num_releases
select
    COUNT(*) as total_releases
from
    releases
where
    major_version = ${params.release}
```

```sql num_minor_releases
select
    COUNT(*) as minor_releases
from
    releases
where
    major_version = ${params.release}
    and is_minor = 1
```

```sql commits_by_type
select
    type,
    COUNT(*) as count
from
    commits
where
    major_version = ${params.release}
    and type is not null
group by
    type
order by
    count desc
```

```sql commits_by_type_no_deps
select
    type,
    COUNT(*) as count
from
    commits
where
    major_version = ${params.release}
    and type is not null
    and is_dependency_update = 0
group by
    type
order by
    count desc
```

```sql top_10_scopes
select
    scope,
    COUNT(*) as count
from
    commits
where
    major_version = ${params.release}
    and scope is not null
    and is_dependency_update = 0
group by
    scope
order by
    count desc
limit
    10
```

```sql top_release_dates
select
    release_date as date,
    COUNT(*) as num_releases
from
    releases
where
    major_version = ${params.release}
    and tag not like '${params.release}.%-next%'
group by
    release_date
order by
    num_releases desc
limit
    3
```

```sql all_release_dates
select
    release_date as date,
    (
        case
            when is_minor == 1 then 'Minor'
            else 'Patch'
        end
    ) as release_type,
    COUNT(*) as num_releases
from
    releases
where
    major_version = ${params.release}
    and tag not like '${params.release}.%-next%'
group by
    release_date,
    release_type
order by
    release_date asc
```

```sql longest_commit_streak
with commit_dates as (
    select
        distinct commit_date
    from
        commits
    where
        major_version = ${params.release}
    order by
        commit_date
),
date_gaps as (
    select
        commit_date,
        LAG(commit_date) OVER (
            order by
                commit_date
        ) as prev_date,
        DATE_DIFF(
            'day',
            LAG(commit_date) OVER (
                order by
                    commit_date
            ),
            commit_date
        ) as day_diff
    from
        commit_dates
),
streak_groups as (
    select
        commit_date,
        SUM(
            case
                when day_diff > 1
                or day_diff is null then 1
                else 0
            end
        ) OVER (
            order by
                commit_date
        ) as streak_id
    from
        date_gaps
),
streaks as (
    select
        streak_id,
        MIN(commit_date) as start_date,
        MAX(commit_date) as end_date,
        DATE_DIFF('day', MIN(commit_date), MAX(commit_date)) + 1 as days
    from
        streak_groups
    group by
        streak_id
)
select
    days,
    start_date,
    end_date
from
    streaks
order by
    days desc
limit
    1;
```
