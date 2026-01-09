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
