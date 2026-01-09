-- name: FindMissingDiscussions :many
select
    *
from
    discussions
where
    number not in (sqlc.slice('numbers'));
