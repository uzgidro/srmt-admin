CREATE TABLE IF NOT EXISTS reservoirs (
                                          id INTEGER PRIMARY KEY AUTOINCREMENT,
                                          name TEXT NOT NULL UNIQUE,
                                          position TEXT
);

CREATE TABLE IF NOT EXISTS level_volume (
                                            id INTEGER PRIMARY KEY AUTOINCREMENT,
                                            level REAL NOT NULL,
                                            volume REAL NOT NULL,
                                            res_id INTEGER NOT NULL,
                                            FOREIGN KEY (res_id) REFERENCES reservoirs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS data (
                                    id INTEGER PRIMARY KEY AUTOINCREMENT,
                                    level REAL NOT NULL,
                                    volume REAL NOT NULL,
                                    "release" REAL,
                                    income REAL,
                                    res_id INTEGER NOT NULL,
                                    date TEXT NOT NULL,
                                    UNIQUE (res_id, date),
                                    FOREIGN KEY (res_id) REFERENCES reservoirs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS indicator_height (
                                                id INTEGER PRIMARY KEY AUTOINCREMENT,
                                                height REAL NOT NULL,
                                                res_id INTEGER NOT NULL,
                                                FOREIGN KEY (res_id) REFERENCES reservoirs(id) ON DELETE CASCADE
);