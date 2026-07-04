CREATE TABLE IF NOT EXISTS app_settings (
    id          BIGSERIAL PRIMARY KEY,
    key         TEXT NOT NULL UNIQUE,
    value       TEXT NOT NULL,
    encrypted   BOOLEAN NOT NULL DEFAULT false,
    active      BOOLEAN NOT NULL DEFAULT true,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO app_settings (key, value, description) VALUES
    ('webhook.stripe.signing_secret', 'whsec_sandbox_replace_me', 'Stripe webhook signing secret'),
    ('webhook.github.signing_secret', 'gh_secret_sandbox_replace_me', 'GitHub webhook secret'),
    ('payments.stripe.api_key', 'sk_test_sandbox_replace_me', 'Stripe secret API key — stored in DB, not env')
ON CONFLICT (key) DO NOTHING;
