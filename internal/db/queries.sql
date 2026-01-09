-- name: FindKnownDiscussions :many
select
    number
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
        answered_by,
        labels
    )
values
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: InsertDiscussionComment :exec
insert into
    discussion_comments (
        discussion_number,
        id,
        created_at,
        updated_at,
        author,
        reply_to
    )
values
    (?, ?, ?, ?, ?, ?);
