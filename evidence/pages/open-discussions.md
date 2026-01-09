---
title: Open Discussions
---

## Open Discussions, by age

```sql open_age
select
    category_name,
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
            and category_name = 'Request Help'
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
    category_name,
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
            and category_name = 'Suggest an Idea'
        group by
            category_name,
            discussions.created_at
        order by
            discussions.created_at asc
    )
group by
    category_name,
    month
```

<BarChart
data={open_age}
series=category_name
x=month
y=num
title="Open Discussions, by creation date"
/>

```sql closed_age
select
    category_name,
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
            and category_name = 'Request Help'
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
    category_name,
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
            and category_name = 'Suggest an Idea'
        group by
            category_name,
            discussions.created_at
        order by
            discussions.created_at asc
    )
group by
    category_name,
    month
```

<BarChart
data={closed_age}
series=category_name
x=month
y=num
title="Closed Discussions, by creation date"
/>

## Open Discussions, with no comments

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


```sql open_no_comments
select
    number,
    ANY_VALUE(title),
    ANY_VALUE(url),
    ANY_VALUE(discussions.created_at),
    ANY_VALUE(discussions.updated_at)
from
    discussions
    left join discussion_comments on discussion_comments.discussion_number = discussions.number
where
    (
        state = 'OPEN'
        or state = 'REOPENED'
    )
    and discussion_comments.discussion_number is null
    and category_name = '${inputs.category_name.value}'
group by
    discussions.number,
    discussions.created_at
order by
    discussions.created_at asc
```

<DataTable data={open_no_comments} />

## Closed without a reply

```sql closed_no_reply
select
    number,
    ANY_VALUE(discussions.author),
    ANY_VALUE(category_name),
    ANY_VALUE(title),
    ANY_VALUE(url),
    ANY_VALUE(discussions.created_at),
    ANY_VALUE(discussions.updated_at)
    -- TODO: closed_by
from
    discussions
    left join discussion_comments on discussion_comments.discussion_number = discussions.number
where
    (
        state != 'OPEN'
        and state != 'REOPENED'
    )
    and discussion_comments.discussion_number is null
group by
    discussions.number,
    discussions.created_at
order by
    discussions.created_at asc
```
