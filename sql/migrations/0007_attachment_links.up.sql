BEGIN;

CREATE TABLE attachment_links (
    attachment_id UUID NOT NULL REFERENCES attachments(id) ON DELETE CASCADE,
    user_id INTEGER NOT NULL REFERENCES users(id),
    resource_type TEXT NOT NULL,
    resource_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (attachment_id, resource_type, resource_id)
);

CREATE INDEX attachment_links_user_resource_idx ON attachment_links(user_id, resource_type, resource_id);
CREATE INDEX attachment_links_attachment_idx ON attachment_links(attachment_id);

COMMIT;
