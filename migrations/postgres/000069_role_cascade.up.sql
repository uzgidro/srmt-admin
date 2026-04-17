INSERT INTO roles (name, description)
VALUES ('cascade', 'Оператор каскада ГЭС — доступ к станциям своего каскада')
ON CONFLICT (name) DO NOTHING;
