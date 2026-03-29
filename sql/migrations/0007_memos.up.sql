BEGIN;

CREATE TABLE memos (
    id UUID PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    content TEXT NOT NULL,
    excerpt TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'active',
    archived_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT memos_state_check CHECK (state IN ('active', 'archived'))
);

CREATE INDEX memos_user_state_updated_idx ON memos(user_id, state, updated_at DESC, id DESC);

CREATE TABLE memo_tags (
    memo_id UUID NOT NULL REFERENCES memos(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    tag TEXT NOT NULL,
    PRIMARY KEY (memo_id, tag)
);

CREATE INDEX memo_tags_user_tag_memo_idx ON memo_tags(user_id, tag, memo_id);

CREATE TABLE memo_relations (
    source_memo_id UUID NOT NULL REFERENCES memos(id) ON DELETE CASCADE,
    target_memo_id UUID NOT NULL REFERENCES memos(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    PRIMARY KEY (source_memo_id, target_memo_id)
);

CREATE INDEX memo_relations_user_target_idx ON memo_relations(user_id, target_memo_id);
CREATE INDEX memo_relations_user_source_idx ON memo_relations(user_id, source_memo_id);

COMMIT;
