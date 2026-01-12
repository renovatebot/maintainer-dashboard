-- name: FindKnownDiscussions :many
select
    number
from
    discussions;

-- name: FindMostRecentlyUpdatedDiscussion :one
select
    updated_at
from
    discussions
limit
    1;

-- name: FindUpdatedTimesForDiscussions :many
select
    number,
    updated_at
from
    discussions
where
    number in (sqlc.slice('numbers'));

-- name: InsertDiscussion :exec
insert into
    discussions (
        number,
        title,
        url,
        state,
        created_at,
        updated_at,
        closed_at,
        author,
        category_name,
        answer_chosen_at,
        answer_chosen_by,
        answered_by,
        labels,
        body,
        upvote_count
    )
values
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) on conflict(number) do
update
set
    title = excluded.title,
    url = excluded.url,
    state = excluded.state,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    closed_at = excluded.closed_at,
    author = excluded.author,
    category_name = excluded.category_name,
    answer_chosen_at = excluded.answer_chosen_at,
    answer_chosen_by = excluded.answer_chosen_by,
    answered_by = excluded.answered_by,
    labels = excluded.labels,
    body = excluded.body,
    upvote_count = excluded.upvote_count;

-- name: InsertDiscussionComment :exec
insert into
    discussion_comments (
        discussion_number,
        id,
        created_at,
        updated_at,
        author,
        reply_to,
        body,
        upvote_count
    )
values
    (?, ?, ?, ?, ?, ?, ?, ?) on conflict(id) do
update
set
    id = excluded.id,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    author = excluded.author,
    reply_to = excluded.reply_to,
    body = excluded.body,
    upvote_count = excluded.upvote_count;
