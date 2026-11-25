---
title: Release trains
---

Statistics about each major version ("release train") of Renovate, and some insights into how many contributions we've had.

```sql all_releases_no_date
-- Co-authored-by: Claude Sonnet 4.5 (GitHub Copilot)
select
    DENSE_RANK() OVER (
        partition by release_train
        order by
            release_date
    ) - 1 as release_number,
    release_train,
    release_date,
    COUNT(*) as releases_on_date
from
    releases
where
    tag not like '%-next%'
    and release_train > 36
group by
    release_train,
    release_date
order by
    release_train desc,
    release_date
```

<BarChart title="# of releases per day of a given major version (last 6)" data={all_releases_no_date} series="release_train" />


```sql train_summary
with rel as (
    select
        release_train,
        min(release_date) filter (where tag not like '%-next%') as first_release,
        max(release_date) filter (where tag not like '%-next%') as last_release,
        count(*) filter (where tag not like '%-next%') as releases
    from
        releases
    where
        release_train is not null
    group by
        release_train
),
com as (
    select
        release_train,
        count(*) as commits,
        avg(is_dependency_update) as dependency_share
    from
        commits
    where
        release_train is not null
    group by
        release_train
)
select
    rel.release_train,
    rel.first_release,
    rel.last_release,
    date_diff('day', rel.first_release, rel.last_release) + 1 as days,
    rel.releases,
    round(rel.releases * 7.0 / (date_diff('day', rel.first_release, rel.last_release) + 1), 1) as releases_per_week,
    com.commits,
    com.dependency_share
from
    rel
    left join com using (release_train)
where
    rel.release_train > 36
order by
    rel.release_train desc
```

Comparing each release train (excluding pre-releases):

<DataTable data={train_summary}>
    <Column id=release_train title="Train" fmt=num0 />
    <Column id=first_release title="First release" fmt="d mmm yyyy" />
    <Column id=last_release title="Last release" fmt="d mmm yyyy" />
    <Column id=days title="Days" />
    <Column id=releases title="Releases" />
    <Column id=releases_per_week title="Releases / week" />
    <Column id=commits title="Commits" />
    <Column id=dependency_share title="Dependency updates" fmt=pct0 />
</DataTable>

```sql major_versions
select
    release_train,
    count(*) filter (where tag not like '%-next%') as num_releases
from
    release_trains.releases
where
    release_train is not null
group by
    release_train
order by
    release_train desc;
```

{#each major_versions as version}

- [{version.release_train}.x ({version.num_releases} releases)](/release-train/{version.release_train}/)

{/each}
