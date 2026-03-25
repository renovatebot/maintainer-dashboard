create table if not exists discussions (
    number integer primary key,
    title text not null,
    url text not null,
    state text check(
        state in (
            -- a custom value to denote a currently open Discussion, as 'OPEN' is the implied state
            'OPEN',
            -- GraphQL entries for DiscussionStateReason
            'RESOLVED',
            'OUTDATED',
            'DUPLICATE',
            'REOPENED'
        )
    ) not null,
    created_at text not null,
    -- updated_at stores the GitHub Discussion.updatedAt value
    -- indicates the last time the Discussion was updated, including comments, but doesn't include a comment being edited
    updated_at text not null,
    closed_at text,
    author text not null,
    category_name text not null,
    answer_chosen_at text,
    answered_by text,
    -- a JSON array of label names
    labels json
);

create table if not exists discussion_comments (
    -- TODO foreign key
    discussion_number integer not null,
    id text primary key,
    created_at text not null,
    updated_at text not null,
    author text not null,
    -- NOTE: NULL for top-level comments (not replies)
    reply_to text
);

create table if not exists categories (
    id text primary key,
    name text not null,
    slug text not null,
    is_answerable INTEGER not null check (is_answerable in (0, 1)),
    updated_at text not null
);

alter table
    discussions
add
    column body text;

alter table
    discussion_comments
add
    column body text;

alter table
    discussions
add
    column upvote_count integer;

alter table
    discussion_comments
add
    column upvote_count integer;

alter table
    discussions
add
    column answer_chosen_by text;
