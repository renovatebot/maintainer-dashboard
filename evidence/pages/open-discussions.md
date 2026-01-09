---
title: Open Discussions
---

## Open Discussions, with no comments

```sql categories
select
    distinct category_name
from
    discussions
```

<Dropdown
name=category_name
data={categories}
value=category_name
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
    state = 'OPEN'
    and discussion_comments.discussion_number is null
    and category_name = '${inputs.category_name.value}'
group by
    discussions.number,
    discussions.created_at
order by
    discussions.created_at asc
```

<DataTable data={open_no_comments} />
