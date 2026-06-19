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

-- name: FindKnownIssues :many
select
    number
from
    issues;

-- name: FindMostRecentlyUpdatedIssue :one
select
    updated_at
from
    issues
order by
    updated_at desc
limit
    1;

-- name: InsertIssue :exec
insert into
    issues (
        number,
        title,
        url,
        state,
        state_reason,
        created_at,
        updated_at,
        closed_at,
        author,
        labels,
        body,
        locked,
        reactions
    )
values
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) on conflict(number) do
update
set
    title = excluded.title,
    url = excluded.url,
    state = excluded.state,
    state_reason = excluded.state_reason,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    closed_at = excluded.closed_at,
    author = excluded.author,
    labels = excluded.labels,
    body = excluded.body,
    locked = excluded.locked,
    reactions = excluded.reactions;

-- name: InsertIssueComment :exec
insert into
    issue_comments (
        issue_number,
        id,
        created_at,
        updated_at,
        author,
        body
    )
values
    (?, ?, ?, ?, ?, ?) on conflict(id) do
update
set
    id = excluded.id,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    author = excluded.author,
    body = excluded.body;

-- name: FindKnownPullRequests :many
select
    number
from
    pull_requests;

-- name: FindMostRecentlyUpdatedPullRequest :one
select
    updated_at
from
    pull_requests
order by
    updated_at desc
limit
    1;

-- name: InsertPullRequest :exec
insert into
    pull_requests (
        number,
        title,
        url,
        state,
        created_at,
        updated_at,
        closed_at,
        merged_at,
        author,
        labels,
        body,
        is_draft,
        head_ref_name,
        base_ref_name,
        review_decision,
        additions,
        deletions,
        changed_files
    )
values
    (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) on conflict(number) do
update
set
    title = excluded.title,
    url = excluded.url,
    state = excluded.state,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at,
    closed_at = excluded.closed_at,
    merged_at = excluded.merged_at,
    author = excluded.author,
    labels = excluded.labels,
    body = excluded.body,
    is_draft = excluded.is_draft,
    head_ref_name = excluded.head_ref_name,
    base_ref_name = excluded.base_ref_name,
    review_decision = excluded.review_decision,
    additions = excluded.additions,
    deletions = excluded.deletions,
    changed_files = excluded.changed_files;
