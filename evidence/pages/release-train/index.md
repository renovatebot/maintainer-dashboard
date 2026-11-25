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
    tag not like '.%-next%'
    and release_train > 36
group by
    release_train,
    release_date
order by
    release_train desc,
    release_date
```

<BarChart title="# of releases per day of a given major version (last 6)" data={all_releases_no_date} series="release_train" />


```sql major_versions
select
    release_train,
    count(*) as num_releases
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
