INSERT INTO app_settings (key, value, description) VALUES
    ('auth.demo.email', 'demo@koro.io', 'Portal demo login email'),
    ('auth.demo.password', 'password', 'Portal demo login password'),
    ('auth.demo.name', 'Alex Rivera', 'Portal demo display name'),
    ('auth.demo.role', 'admin', 'Portal demo user role')
ON CONFLICT (key) DO NOTHING;
