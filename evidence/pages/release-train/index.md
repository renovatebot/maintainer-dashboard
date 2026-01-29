---
title: Release trains
---

Statistics about each major version ("release train") of Renovate, and some insights into how many contributions we've had.

```sql all_releases_no_date
-- Co-authored-by: Claude Sonnet 4.5 (GitHub Copilot)
select
    DENSE_RANK() OVER (
        partition by major_version
        order by
            release_date
    ) - 1 as release_number,
    major_version,
    release_date,
    COUNT(*) as releases_on_date
from
    releases
where
    tag not like '.%-next%'
    and major_version > 36
group by
    major_version,
    release_date
order by
    major_version desc,
    release_date
```

<BarChart title="# of releases per day of a given major version (last 6)" data={all_releases_no_date} series="major_version" />


```sql major_versions
select
    distinct major_version,
    count(*) as num_releases
from
    release_trains.releases
where
    major_version is not null
group by
    major_version
order by
    major_version desc;
```

{#each major_versions as version}

- [{version.major_version}.x ({version.num_releases} releases)](/release-train/{version.major_version}/)

{/each}
