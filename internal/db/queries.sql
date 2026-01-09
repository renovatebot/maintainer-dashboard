-- name: FindKnownDiscussions :many
select
    number
from
    discussions
where
    number in (sqlc.slice('numbers'));
