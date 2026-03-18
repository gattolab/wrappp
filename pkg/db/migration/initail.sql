create table short_urls
(
    id                  bigserial primary key,
    short_code          varchar(64) not null,
    original_url        text not null,
    created_at          timestamptz not null default now(),
    expire_at           timestamptz,
    last_accessed_at    timestamptz,
    is_active           boolean not null default true,
    click_count         bigint not null default 0,
    password_hash       text,
    utm                 jsonb
);

create unique index short_urls_short_code_uindex
    on short_urls (short_code);

create index short_urls_expire_at_index
    on short_urls (expire_at);

create index short_urls_last_accessed_at_index
    on short_urls (last_accessed_at);

create index short_urls_active_index
    on short_urls (short_code)
    where is_active = true;

alter table short_urls owner to wrappp;

-- Function: manually call or invoke from app scheduler to purge stale/expired rows
CREATE OR REPLACE FUNCTION cleanup_short_urls() RETURNS void
LANGUAGE sql AS $$
    DELETE FROM short_urls
    WHERE
        -- expired
        (expire_at IS NOT NULL AND expire_at <= NOW())
        OR
        -- never accessed and created > 30 days ago
        (last_accessed_at IS NULL AND created_at <= NOW() - INTERVAL '30 days')
        OR
        -- last accessed > 30 days ago
        (last_accessed_at IS NOT NULL AND last_accessed_at <= NOW() - INTERVAL '30 days');
$$;

