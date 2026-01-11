---
title: Open Discussions
---

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

## Open Discussions, by category

```sql open_request_help
select
    number,
    ANY_VALUE(title) as title,
    ANY_VALUE(url) as url,
    ANY_VALUE(discussions.created_at) as created_at,
    ANY_VALUE(discussions.updated_at),
    count(discussion_comments.discussion_number)
from
    discussions
    left join discussion_comments on discussion_comments.discussion_number = discussions.number
where
    (
        state = 'OPEN'
        or state = 'REOPENED'
    )
    and category_name = 'Request Help'
group by
    discussions.number,
    discussions.created_at
order by
    discussions.created_at desc
```

<DataTable data={open_request_help} />

```sql open_suggest_idea
select
    number,
    ANY_VALUE(title) as title,
    ANY_VALUE(url) as url,
    ANY_VALUE(discussions.created_at) as created_at,
    ANY_VALUE(discussions.updated_at),
    count(discussion_comments.discussion_number)
from
    discussions
    left join discussion_comments on discussion_comments.discussion_number = discussions.number
where
    (
        state = 'OPEN'
        or state = 'REOPENED'
    )
    and category_name = 'Suggest an Idea'
group by
    discussions.number,
    discussions.created_at
order by
    discussions.created_at desc
```

<DataTable data={open_suggest_idea} />

```sql open_mend_hosted
select
    number,
    ANY_VALUE(title) as title,
    ANY_VALUE(url) as url,
    ANY_VALUE(discussions.created_at) as created_at,
    ANY_VALUE(discussions.updated_at),
    count(discussion_comments.discussion_number)
from
    discussions
    left join discussion_comments on discussion_comments.discussion_number = discussions.number
where
    (
        state = 'OPEN'
        or state = 'REOPENED'
    )
    and category_name = 'Mend Hosted Request'
group by
    discussions.number,
    discussions.created_at
order by
    discussions.created_at desc
```

<DataTable data={open_mend_hosted} />

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

## Closed unanswered

```sql closed_unanswered
select
    number,
    (discussions.author),
    (category_name),
    (title),
    (url),
    (discussions.created_at),
    (discussions.updated_at),
    answer_chosen_at
    -- TODO: closed_by
from
    discussions
where
    (
        answer_chosen_at is null
        or answer_chosen_at = '1970-01-01'
    )
    and category_name = 'Request Help'
    and (
        state != 'OPEN'
        and state != 'REOPENED'
    )
order by
    discussions.created_at asc
```

## Bumped after some time

```sql bumped_after_many_days_open
-- Co-authored-by: gpt-4.1 (GitHub Copilot)
with ranked_comments as (
    select
        discussions.number,
        discussions.title,
        discussions.url,
        discussions.created_at,
        discussion_comments.updated_at,
        discussion_comments.author,
        ROW_NUMBER() OVER (
            partition BY discussions.number
            order by
                discussion_comments.updated_at desc
        ) as rn
    from
        discussions
        inner join discussion_comments on discussion_comments.discussion_number = discussions.number
    where
        state in ('OPEN', 'REOPENED')
)
select
    updated_at,
    author as "Last commenter",
    title,
    url,
    date_diff('day', created_at, updated_at) as "Days after creation",
from
    ranked_comments
where
    rn = 1
order by
    "Days after creation" desc,
    updated_at desc
limit
    50
```

```sql bumped_after_many_days_closed
-- Co-authored-by: gpt-4.1 (GitHub Copilot)
with ranked_comments as (
    select
        discussions.number,
        discussions.title,
        discussions.url,
        discussions.created_at,
        discussion_comments.updated_at,
        discussion_comments.author,
        ROW_NUMBER() OVER (
            partition BY discussions.number
            order by
                discussion_comments.updated_at desc
        ) as rn
    from
        discussions
        inner join discussion_comments on discussion_comments.discussion_number = discussions.number
    where
        state not in ('OPEN', 'REOPENED')
)
select
    updated_at,
    author as "Last commenter",
    title,
    url,
    date_diff('day', created_at, updated_at) as "Days after creation",
from
    ranked_comments
where
    rn = 1
order by
    "Days after creation" desc,
    updated_at desc
limit
    50
```

<DataTable data={bumped_after_many_days_open} title="Open Discussions, bumped after some time" />
<DataTable data={bumped_after_many_days_closed} title="Closed Discussions, bumped after some time" />
