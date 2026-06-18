CREATE TABLE links (
    id UUID PRIMARY KEY,
    code TEXT NOT NULL,
    original_url TEXT NOT NULL,
    is_custom BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    disabled_at TIMESTAMPTZ NULL,
    CONSTRAINT links_code_unique UNIQUE (code)
);

CREATE TABLE clicks (
    id UUID PRIMARY KEY,
    link_id UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_agent TEXT NOT NULL,
    referer TEXT NULL,
    ip INET NULL
);

CREATE INDEX clicks_link_id_clicked_at_idx ON clicks (link_id, clicked_at DESC);
CREATE INDEX clicks_link_id_user_agent_idx ON clicks (link_id, user_agent);

