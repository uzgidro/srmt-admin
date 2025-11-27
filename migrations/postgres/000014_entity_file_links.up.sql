-- Incident file links
CREATE TABLE IF NOT EXISTS incident_file_links (
    incident_id BIGINT NOT NULL REFERENCES Incidents(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (incident_id, file_id)
);

CREATE INDEX idx_incident_file_links_incident ON incident_file_links(incident_id);
CREATE INDEX idx_incident_file_links_file ON incident_file_links(file_id);

-- Discharge file links
CREATE TABLE IF NOT EXISTS discharge_file_links (
    discharge_id BIGINT NOT NULL REFERENCES idle_water_discharges(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (discharge_id, file_id)
);

CREATE INDEX idx_discharge_file_links_discharge ON discharge_file_links(discharge_id);
CREATE INDEX idx_discharge_file_links_file ON discharge_file_links(file_id);

-- Shutdown file links
CREATE TABLE IF NOT EXISTS shutdown_file_links (
    shutdown_id BIGINT NOT NULL REFERENCES Shutdowns(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (shutdown_id, file_id)
);

CREATE INDEX idx_shutdown_file_links_shutdown ON shutdown_file_links(shutdown_id);
CREATE INDEX idx_shutdown_file_links_file ON shutdown_file_links(file_id);

-- Visit file links
CREATE TABLE IF NOT EXISTS visit_file_links (
    visit_id BIGINT NOT NULL REFERENCES Visits(id) ON DELETE CASCADE,
    file_id BIGINT NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (visit_id, file_id)
);

CREATE INDEX idx_visit_file_links_visit ON visit_file_links(visit_id);
CREATE INDEX idx_visit_file_links_file ON visit_file_links(file_id);
