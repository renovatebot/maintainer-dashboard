---
title: Issues
---

## Open Issues, by age

```sql open_age
select
    date_trunc('month', issues.created_at) as month,
    count(1) as num
from
    issues
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
title="Open Issues, by creation date"
/>

```sql closed_age
select
    date_trunc('month', issues.created_at) as month,
    count(1) as num
from
    issues
where
    state = 'CLOSED'
group by
    month
order by
    month asc
```

<BarChart
data={closed_age}
x=month
y=num
title="Closed Issues, by creation date"
/>

```sql open_close_age
select
    state,
    date_trunc('month', issues.created_at) as month,
    count(1) as num
from
    issues
group by
    state,
    month
order by
    month asc
```

<BarChart
data={open_close_age}
series=state
x=month
y=num
title="Open/Closed Issues, by creation date"
/>

