CREATE TABLE IF NOT EXISTS public.users (
    id          text        PRIMARY KEY,
    email       text        NOT NULL UNIQUE,
    full_name   text        NOT NULL DEFAULT '',
    avatar_url  text        NOT NULL DEFAULT '',
    is_platform_admin bool  NOT NULL DEFAULT false,
    locale      text        NOT NULL DEFAULT 'en',
    created_at  timestamptz NOT NULL DEFAULT now()
);
