---
---
# Renovate {params.release} release train stats

```sql releases_over_release_train
-- Co-authored-by: Claude Sonnet 4.5 (GitHub Copilot)
select
    DENSE_RANK() OVER (
        order by
            release_date
    ) - 1 as release_number,
    (
        case
            when is_minor == 1 then 'Minor'
            else 'Patch'
        end
    ) as release_type,
    release_date,
    COUNT(*) as releases_on_date
from
    releases
where
    release_train = ${params.release}
    and tag not like '${params.release}.%-next%'
group by
    release_date,
    is_minor
order by
    release_date,
    release_type;
```

<BarChart data={releases_over_release_train} series="release_type" title="# of releases over each day of the release train" />

In Renovate {params.release}.x, there were:

- <Value data={num_commits} /> commits
- <Value data={num_releases} /> releases
- <Value data={num_minor_releases} /> minor releases
- <Value data={longest_commit_streak} column=days /> consecutive days of commits

And had the following release cadence:

<BarChart data={all_release_dates} series="release_type">
    <ReferenceLine data={major_releases} x=date label=tag labelPosition=aboveEnd color=negative lineType=solid />
</BarChart>

Commits per month, split by whether they were dependency updates:

<BarChart data={commits_by_month} x=month y=commits series=kind sort=false />

And the weekdays releases went out on:

<BarChart data={releases_by_weekday} x=weekday y=num_releases sort=false />

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
    release_train = ${params.release}
```


```sql num_commits
select
    COUNT(*) as total_commits
from
    commits
where
    release_train = ${params.release}
```

```sql num_releases
select
    COUNT(*) as total_releases
from
    releases
where
    release_train = ${params.release}
```

```sql num_minor_releases
select
    COUNT(*) as minor_releases
from
    releases
where
    release_train = ${params.release}
    and is_minor = 1
```

```sql commits_by_type
select
    type,
    COUNT(*) as count
from
    commits
where
    release_train = ${params.release}
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
    release_train = ${params.release}
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
    release_train = ${params.release}
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
    release_train = ${params.release}
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
            when minor_version = 0
            and patch_version = 0 then 'Major'
            when is_minor == 1 then 'Minor'
            else 'Patch'
        end
    ) as release_type,
    COUNT(*) as num_releases
from
    releases
where
    release_train = ${params.release}
    and tag not like '%-next%'
group by
    release_date,
    release_type
order by
    release_date asc
```

```sql major_releases
select
    release_date as date,
    tag
from
    releases
where
    release_train = ${params.release}
    and minor_version = 0
    and patch_version = 0
    and tag not like '%-next%'
order by
    release_date
```

```sql commits_by_month
select
    strftime(date_trunc('month', commit_date), '%b %Y') as month,
    date_trunc('month', commit_date) as month_start,
    (
        case
            when is_dependency_update = 1 then 'Dependency update'
            else 'Other'
        end
    ) as kind,
    COUNT(*) as commits
from
    commits
where
    release_train = ${params.release}
group by
    month,
    month_start,
    kind
order by
    month_start
```

```sql releases_by_weekday
select
    isodow(release_date) as weekday_number,
    dayname(release_date) as weekday,
    COUNT(*) as num_releases
from
    releases
where
    release_train = ${params.release}
    and tag not like '%-next%'
group by
    weekday_number,
    weekday
order by
    weekday_number
```

```sql longest_commit_streak
with commit_dates as (
    select
        distinct commit_date
    from
        commits
    where
        release_train = ${params.release}
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
