---
title: Pull Requests
---

```sql pr_summary
select
    state,
    count(1) as num
from
    pull_requests
group by
    state
order by
    state asc
```

<DataTable data={pr_summary} />

## Open Pull Requests

```sql open_prs
select
    number,
    title,
    author,
    head_ref_name,
    base_ref_name,
    is_draft,
    review_decision,
    additions,
    deletions,
    changed_files,
    created_at,
    updated_at,
    url
from
    pull_requests
where
    state = 'OPEN'
order by
    updated_at desc
```

<DataTable data={open_prs} link=url />

## Open PRs by age

```sql open_age
select
    date_trunc('month', pull_requests.created_at) as month,
    count(1) as num
from
    pull_requests
where
    state = 'OPEN'
group by
    month
order by
    month asc
```

<BarChart
data={open_age}
x=month
y=num
title="Open PRs, by creation date"
/>

## Open/Closed/Merged PRs by creation date

```sql state_age
select
    state,
    date_trunc('month', pull_requests.created_at) as month,
    count(1) as num
from
    pull_requests
group by
    state,
    month
order by
    month asc
```

<BarChart
data={state_age}
series=state
x=month
y=num
title="PRs by state and creation date"
/>
