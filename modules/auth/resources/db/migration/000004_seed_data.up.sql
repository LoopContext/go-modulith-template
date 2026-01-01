-- Seed Roles
INSERT INTO roles (id, name) VALUES
('role_01j6z1p3m7r2b8v8x0w1m4p2r1', 'admin'),
('role_01j6z1p3m7r2b8v8x0w1m4p2r2', 'user')
ON CONFLICT (id) DO NOTHING;

-- Seed Permissions
INSERT INTO permissions (id, name, resource, action) VALUES
('perm_01j6z1p3m7r2b8v8x0w1m4p2rk', 'users:read', 'users', 'read'),
('perm_01j6z1p3m7r2b8v8x0w1m4p2rm', 'users:write', 'users', 'write'),
('perm_01j6z1p3m7r2b8v8x0w1m4p2rn', 'auth:debug', 'auth', 'debug')
ON CONFLICT (id) DO NOTHING;

-- Link Roles and Permissions
INSERT INTO role_permissions (role_id, permission_id) VALUES
('role_01j6z1p3m7r2b8v8x0w1m4p2r1', 'perm_01j6z1p3m7r2b8v8x0w1m4p2rk'),
('role_01j6z1p3m7r2b8v8x0w1m4p2r1', 'perm_01j6z1p3m7r2b8v8x0w1m4p2rm'),
('role_01j6z1p3m7r2b8v8x0w1m4p2r1', 'perm_01j6z1p3m7r2b8v8x0w1m4p2rn'),
('role_01j6z1p3m7r2b8v8x0w1m4p2r2', 'perm_01j6z1p3m7r2b8v8x0w1m4p2rk')
ON CONFLICT DO NOTHING;

-- Seed Test Users
INSERT INTO users (id, email, phone) VALUES
('user_01j6z1p3m7r2b8v8x0w1m4p2tx', 'admin@example.com', '+10000000000'),
('user_01j6z1p3m7r2b8v8x0w1m4p2tz', 'user@example.com', '+10000000001')
ON CONFLICT (id) DO NOTHING;

-- Assign Roles to Users
INSERT INTO user_roles (user_id, role_id) VALUES
('user_01j6z1p3m7r2b8v8x0w1m4p2tx', 'role_01j6z1p3m7r2b8v8x0w1m4p2r1'),
('user_01j6z1p3m7r2b8v8x0w1m4p2tz', 'role_01j6z1p3m7r2b8v8x0w1m4p2r2')
ON CONFLICT DO NOTHING;

