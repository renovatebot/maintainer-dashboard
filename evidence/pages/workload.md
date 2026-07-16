---
title: Maintainer Workload
---

Volume of new work and reply activity, over the last 16 full weeks (partial current week excluded). Pull request counts exclude `renovate[bot]`, which accounts for roughly half of all PRs and would otherwise dominate the chart.

## New items per week

```sql summary
with pr as (
    select
        count(1) as n
    from
        pull_requests
    where
        author != 'renovate[bot]'
        and created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
        and created_at::timestamp < date_trunc('week', current_date)
),
disc as (
    select
        count(1) as n
    from
        discussions
    where
        created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
        and created_at::timestamp < date_trunc('week', current_date)
),
iss as (
    select
        count(1) as n
    from
        issues
    where
        created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
        and created_at::timestamp < date_trunc('week', current_date)
)
select
    round(pr.n / 16.0, 1) as avg_pr_week,
    round(disc.n / 16.0, 1) as avg_disc_week,
    round(iss.n / 16.0, 1) as avg_issue_week,
    round((pr.n + disc.n + iss.n) / 16.0, 1) as avg_combined_week
from
    pr,
    disc,
    iss
```

<Grid cols=4>
<BigValue data={summary} value=avg_pr_week title="Avg new PRs / week" />
<BigValue data={summary} value=avg_disc_week title="Avg new discussions / week" />
<BigValue data={summary} value=avg_issue_week title="Avg new issues / week" />
<BigValue data={summary} value=avg_combined_week title="Combined new items / week" />
</Grid>

```sql weekly_items
select
    'Pull Requests' as type,
    date_trunc('week', created_at::timestamp) as week,
    count(1) as num
from
    pull_requests
where
    author != 'renovate[bot]'
    and created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
    and created_at::timestamp < date_trunc('week', current_date)
group by
    week
union
all
select
    'Discussions' as type,
    date_trunc('week', created_at::timestamp) as week,
    count(1) as num
from
    discussions
where
    created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
    and created_at::timestamp < date_trunc('week', current_date)
group by
    week
union
all
select
    'Issues' as type,
    date_trunc('week', created_at::timestamp) as week,
    count(1) as num
from
    issues
where
    created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
    and created_at::timestamp < date_trunc('week', current_date)
group by
    week
order by
    week
```

<BarChart
data={weekly_items}
series=type
x=week
y=num
title="New items per week"
/>

## Average by day of week

```sql dow_items
select
    type,
    day_num,
    day,
    round(count(1) / 16.0, 2) as avg_per_week
from
    (
        select
            'Pull Requests' as type,
            isodow(created_at::timestamp) as day_num,
            dayname(created_at::timestamp) as day
        from
            pull_requests
        where
            author != 'renovate[bot]'
            and created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
            and created_at::timestamp < date_trunc('week', current_date)
        union
        all
        select
            'Discussions' as type,
            isodow(created_at::timestamp) as day_num,
            dayname(created_at::timestamp) as day
        from
            discussions
        where
            created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
            and created_at::timestamp < date_trunc('week', current_date)
        union
        all
        select
            'Issues' as type,
            isodow(created_at::timestamp) as day_num,
            dayname(created_at::timestamp) as day
        from
            issues
        where
            created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
            and created_at::timestamp < date_trunc('week', current_date)
    )
group by
    type,
    day_num,
    day
order by
    day_num,
    type
```

<BarChart
data={dow_items}
series=type
x=day
y=avg_per_week
sort=false
title="Average new items by day of week"
/>

## Comment activity

Replies added to discussions and issues &mdash; typically outweighs the volume of brand-new items above, since most maintainer effort goes into responding rather than triaging new threads.

```sql comment_summary
with dc as (
    select
        count(1) as n
    from
        discussion_comments
    where
        created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
        and created_at::timestamp < date_trunc('week', current_date)
),
ic as (
    select
        count(1) as n
    from
        issue_comments
    where
        created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
        and created_at::timestamp < date_trunc('week', current_date)
)
select
    round(dc.n / 16.0, 1) as avg_disc_comment_week,
    round(ic.n / 16.0, 1) as avg_issue_comment_week,
    round((dc.n + ic.n) / 16.0, 1) as avg_combined_comment_week
from
    dc,
    ic
```

<Grid cols=3>
<BigValue data={comment_summary} value=avg_disc_comment_week title="Avg discussion comments / week" />
<BigValue data={comment_summary} value=avg_issue_comment_week title="Avg issue comments / week" />
<BigValue data={comment_summary} value=avg_combined_comment_week title="Combined comments / week" />
</Grid>

```sql weekly_comments
select
    'Discussion comments' as type,
    date_trunc('week', created_at::timestamp) as week,
    count(1) as num
from
    discussion_comments
where
    created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
    and created_at::timestamp < date_trunc('week', current_date)
group by
    week
union
all
select
    'Issue comments' as type,
    date_trunc('week', created_at::timestamp) as week,
    count(1) as num
from
    issue_comments
where
    created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
    and created_at::timestamp < date_trunc('week', current_date)
group by
    week
order by
    week
```

<BarChart
data={weekly_comments}
series=type
x=week
y=num
title="Comments added per week"
/>

```sql dow_comments
select
    type,
    day_num,
    day,
    round(count(1) / 16.0, 2) as avg_per_week
from
    (
        select
            'Discussion comments' as type,
            isodow(created_at::timestamp) as day_num,
            dayname(created_at::timestamp) as day
        from
            discussion_comments
        where
            created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
            and created_at::timestamp < date_trunc('week', current_date)
        union
        all
        select
            'Issue comments' as type,
            isodow(created_at::timestamp) as day_num,
            dayname(created_at::timestamp) as day
        from
            issue_comments
        where
            created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
            and created_at::timestamp < date_trunc('week', current_date)
    )
group by
    type,
    day_num,
    day
order by
    day_num,
    type
```

<BarChart
data={dow_comments}
series=type
x=day
y=avg_per_week
sort=false
title="Average comments by day of week"
/>

## Discussion comments by category

```sql category_totals
select
    d.category_name as category,
    count(1) as num
from
    discussion_comments dc
    join discussions d on dc.discussion_number = d.number
where
    dc.created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
    and dc.created_at::timestamp < date_trunc('week', current_date)
group by
    category
order by
    num desc
```

<BarChart
data={category_totals}
x=category
y=num
swapXY=true
title="Discussion comments by category"
/>

```sql category_weekly
select
    case
        d.category_name
        when 'Request Help' then 'Request Help'
        when 'Suggest an Idea' then 'Suggest an Idea'
        else 'Other'
    end as category,
    date_trunc('week', dc.created_at::timestamp) as week,
    count(1) as num
from
    discussion_comments dc
    join discussions d on dc.discussion_number = d.number
where
    dc.created_at::timestamp >= date_trunc('week', current_date) - interval 16 week
    and dc.created_at::timestamp < date_trunc('week', current_date)
group by
    category,
    week
order by
    week
```

<BarChart
data={category_weekly}
series=category
x=week
y=num
title="Comments per week by category"
/>
