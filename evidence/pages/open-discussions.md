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
    ANY_VALUE(title) as title,
    ANY_VALUE(url) as url,
    ANY_VALUE(discussions.created_at) as created_at,
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

## By upvote count

```sql request_help_by_upvote
select
    discussions.upvote_count,
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
    discussions.upvote_count
order by
    discussions.upvote_count desc
```

<DataTable data={request_help_by_upvote} />

## No reply since a GitHub Actions bot

```sql no_reply_or_update_since_bot
-- Co-authored-by: gpt-4.1 (GitHub Copilot)
select
    discussions.category_name || json_array(
        (
            select
                list(value)
            from
                json_each(labels)
            where
                -- NOTE: json_each returns raw JSON values, so string entries include surrounding quotes
                value like '"auto:%'
        )
    ) as series,
    month,
    count(1) as num,
from
    (
        select
            discussion_comments.discussion_number,
            discussion_comments.author,
            discussion_comments.created_at,
            date_trunc('month', discussion_comments.created_at) as month,
            row_number() over (
                partition by discussion_comments.discussion_number
                order by
                    discussion_comments.updated_at asc -- not desc
            ) as rn
        from
            discussion_comments
    ) dc
    inner join discussions on discussions.number = dc.discussion_number
where
    dc.rn = 1
    and dc.author = 'github-actions[bot]'
    and (
        state = 'OPEN'
        or state = 'REOPENED'
    )
    -- and hasn't been updated since
    and discussions.updated_at <= dc.created_at
group by
    series,
    month,
    labels,
```

<BarChart
data={no_reply_or_update_since_bot}
series=series
x=month
y=num
title="Open, but no answer since a `github-actions[bot]` reply"
/>

## Updated since a GitHub Actions bot

```sql updated_since_bot
-- Co-authored-by: gpt-4.1 (GitHub Copilot)
select
    url,
    discussions.updated_at,
    dc.created_at,
from
    (
        select
            discussion_comments.discussion_number,
            discussion_comments.author,
            discussion_comments.created_at,
            date_trunc('month', discussion_comments.created_at) as month,
            row_number() over (
                partition by discussion_comments.discussion_number
                order by
                    discussion_comments.updated_at asc -- not desc
            ) as rn
        from
            discussion_comments
    ) dc
    inner join discussions on discussions.number = dc.discussion_number
where
    dc.rn = 1
    and dc.author = 'github-actions[bot]'
    and (
        state = 'OPEN'
        or state = 'REOPENED'
    )
    and discussions.updated_at > dc.created_at
```

<DataTable data={updated_since_bot} />

## Needing maintainer input

```sql needs_maintainer_input
select
    number,
    ANY_VALUE(title) as title,
    ANY_VALUE(url) as url,
    ANY_VALUE(discussions.created_at) as created_at,
    ANY_VALUE(discussions.updated_at),
    count(discussion_comments.discussion_number)
from
    discussions
    left join discussion_comments on discussion_comments.discussion_number = discussions.number,
    json_each(discussions.labels)
where
    (
        state = 'OPEN'
        or state = 'REOPENED'
    )
    -- NOTE: the JSON value is the quoted string
    and json_each.value = '"maintainer-input-needed"'
group by
    discussions.number,
    discussions.created_at
order by
    discussions.created_at asc
```


<DataTable data={needs_maintainer_input} />

## Awaiting `@jamietanna`'s reply

```sql awaiting_my_reply
-- Co-authored-by: Claude Sonnet 5 (Claude Code)
with my_last_comment as (
    select
        discussion_number,
        max(created_at) as my_last_at
    from
        discussion_comments
    where
        author = 'jamietanna'
    group by
        discussion_number
),
replies_after_me as (
    select
        discussion_comments.discussion_number,
        discussion_comments.created_at,
        discussion_comments.author,
        row_number() over (
            partition by discussion_comments.discussion_number
            order by
                discussion_comments.created_at desc
        ) as rn
    from
        discussion_comments
        inner join my_last_comment on my_last_comment.discussion_number = discussion_comments.discussion_number
    where
        discussion_comments.author != 'jamietanna'
        and discussion_comments.created_at > my_last_comment.my_last_at
)
select
    discussions.number,
    discussions.title,
    discussions.url,
    discussions.category_name,
    my_last_comment.my_last_at,
    replies_after_me.created_at as last_reply_at,
    replies_after_me.author as last_replier,
    date_diff(
        'day',
        cast(replies_after_me.created_at as date),
        current_date
    ) as days_waiting
from
    replies_after_me
    inner join my_last_comment on my_last_comment.discussion_number = replies_after_me.discussion_number
    inner join discussions on discussions.number = replies_after_me.discussion_number
where
    replies_after_me.rn = 1
    and discussions.state in ('OPEN', 'REOPENED')
order by
    replies_after_me.created_at desc
```

<DataTable data={awaiting_my_reply} title="Open discussions I replied to, where someone has replied since" />
