---
title: Welcome to Evidence
---
```sql num_discussions
select
    count(*)
from
    discussions
```

```sql num_comments
select
    count(*)
from
    discussion_comments
```

## Discussions state per category

```sql categories
select
    distinct category_name,
    category_name || ' (' || count(1) || ' total)' as label
from
    discussions
group by
    category_name
```

<Dropdown
name=category_name
data={categories}
value=category_name
label=label
/>

```sql states
select
    state,
    count(1),
from
    discussions
where
    category_name = '${inputs.category_name.value}'
group by
    state
```

<BarChart data={states} title={inputs.category_name.value} />

## Open/close

```sql closed_age_pct
-- co-authored-by: GPT-4.1
select
    closed_percent,
    month
from
    (
        select
            month,
            sum(
                case
                    when state in ('OPEN', 'REOPENED') then 1
                    else 0
                end
            ) as open_count,
            sum(
                case
                    when state not in ('OPEN', 'REOPENED') then 1
                    else 0
                end
            ) as closed_count,
            round(
                100.0 * sum(
                    case
                        when state in ('OPEN', 'REOPENED') then 1
                        else 0
                    end
                ) / count(1),
                2
            ) as open_percent,
            round(
                100.0 * sum(
                    case
                        when state not in ('OPEN', 'REOPENED') then 1
                        else 0
                    end
                ) / count(1),
                2
            ) as closed_percent
        from
            (
                select
                    category_name,
                    date_trunc('month', discussions.created_at) as month,
                    state
                from
                    discussions
                where
                    category_name = '${inputs.category_name.value}'
            ) as sub
        group by
            month
        order by
            month asc
    )
```

<BarChart
  data={closed_age_pct}
  x=month
  y=closed_percent
  title="% closed per month"
  yMax=100
/>

```sql closed_age
select
    'Open' as series,
    month,
    count(1) as num
from
    (
        select
            category_name,
            date_trunc('month', discussions.created_at) as month,
            count(1)
        from
            discussions
        where
            (
                state = 'OPEN'
                or state = 'REOPENED'
            )
            and category_name = '${inputs.category_name.value}'
        group by
            category_name,
            discussions.created_at
        order by
            discussions.created_at asc
    )
group by
    month
union
select
    'Closed' as series,
    month,
    count(1) as num
from
    (
        select
            category_name,
            date_trunc('month', discussions.created_at) as month,
            count(1)
        from
            discussions
        where
            (
                state != 'OPEN'
                and state != 'REOPENED'
            )
            and category_name = '${inputs.category_name.value}'
        group by
            category_name,
            discussions.created_at
        order by
            discussions.created_at asc
    )
group by
    month
```


<BarChart
data={closed_age}
series=series
x=month
y=num
title="Open/closed stats for Discussions, by creation date"
/>

##

```sql category_open_close_over_time
select
    'Opened' as series,
    month,
    count(1) as num
from
    (
        select
            category_name,
            date_trunc('month', discussions.created_at) as month,
            count(1)
        from
            discussions
        where
            category_name = '${inputs.category_name.value}'
            -- category_name = 'Request Help'
        group by
            category_name,
            discussions.created_at
        order by
            discussions.created_at asc
    )
group by
    category_name,
    month
union
select
    'Closed' as series,
    month,
    count(1) as num
from
    (
        select
            category_name,
            date_trunc('month', discussions.closed_at) as month,
            count(1)
        from
            discussions
        where
            category_name = '${inputs.category_name.value}'
            -- category_name = 'Request Help'
            and closed_at != '1970-01-01'
        group by
            category_name,
            discussions.closed_at
        order by
            discussions.closed_at asc
    )
group by
    category_name,
    month
```

<BarChart
data={category_open_close_over_time}
series=series
x=month
y=num
title={"Open/Close stats for '" + inputs.category_name.value + "' over time"}
/>
