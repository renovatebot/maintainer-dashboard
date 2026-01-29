---
title: Release trains
---

Statis **??**

```sql major_versions
select
    distinct major_version
from
    release_trains.releases
where
    major_version is not null
order by
    major_version desc;
```

{#each major_versions as version}

- [{version.major_version}](/release-train/{version.major_version}/)

{/each}
