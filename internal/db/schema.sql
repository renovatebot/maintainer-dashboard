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

create table if not exists issues (
    number integer primary key,
    title text not null,
    url text not null,
    state text check(state in ('OPEN', 'CLOSED')) not null,
    -- GraphQL entries for IssueStateReason: COMPLETED, NOT_PLANNED, REOPENED, or null
    state_reason text,
    created_at text not null,
    -- updated_at stores the GitHub Issue.updatedAt value
    -- as with discussions, this does not bump when an existing comment is edited
    updated_at text not null,
    closed_at text,
    author text not null,
    -- a JSON array of label names
    labels json,
    body text,
    locked integer not null default 0 check(locked in (0, 1)),
    -- a JSON object keyed by reaction content (e.g. {"THUMBS_UP": 5, "HEART": 2})
    reactions json
);

create table if not exists issue_comments (
    -- TODO foreign key
    issue_number integer not null,
    id text primary key,
    created_at text not null,
    updated_at text not null,
    author text not null,
    body text
);

create table if not exists pull_requests (
    number integer primary key,
    title text not null,
    url text not null,
    state text check(state in ('OPEN', 'CLOSED', 'MERGED')) not null,
    created_at text not null,
    -- updated_at stores the GitHub PullRequest.updatedAt value
    updated_at text not null,
    closed_at text,
    merged_at text,
    author text not null,
    -- a JSON array of label names
    labels json,
    body text,
    is_draft integer not null default 0 check(is_draft in (0, 1)),
    head_ref_name text not null,
    base_ref_name text not null,
    -- GraphQL PullRequestReviewDecision: APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, or null
    review_decision text,
    additions integer not null default 0,
    deletions integer not null default 0,
    changed_files integer not null default 0
);
