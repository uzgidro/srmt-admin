CREATE TABLE invest_active_projects (
    id SERIAL PRIMARY KEY,
    category VARCHAR(255) NOT NULL,
    project_name TEXT NOT NULL,
    foreign_partner TEXT,
    implementation_period VARCHAR(255),
    capacity_mw NUMERIC(10, 2),
    production_mln_kwh NUMERIC(15, 2),
    cost_mln_usd NUMERIC(15, 2),
    status_text TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_invest_active_projects_category ON invest_active_projects(category);
CREATE INDEX idx_invest_active_projects_project_name ON invest_active_projects(project_name);
