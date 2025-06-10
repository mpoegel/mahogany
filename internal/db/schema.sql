CREATE TABLE devices (
    id       INTEGER PRIMARY KEY,
    hostname text NOT NULL UNIQUE
);

CREATE TABLE packages (
    id          INTEGER PRIMARY KEY,
    name        text NOT NULL UNIQUE,
    install_cmd text NOT NULL,
    update_cmd  text NOT NULL,
    remove_cmd  text
);

CREATE TABLE assets (
    id         INTEGER PRIMARY KEY,
    device_id  INTEGER NOT NULL,
    package_id INTEGER NOT NULL,
    source_url text,
    version    text,

    FOREIGN KEY(device_id) REFERENCES devices(id),
    FOREIGN KEY(package_id) REFERENCES packages(id)
);

CREATE TABLE settings (
    id    INTEGER PRIMARY KEY,
    name  text NOT NULL,
    value text NOT NULL
);

INSERT INTO settings (name, value)
VALUES ("WatchtowerAddr", "localhost:8080"),
       ("WatchtowerToken", ""),
       ("WatchtowerTimeout", "3s"),
       ("RegistryAddr", "localhost:5000"),
       ("RegistryTimeout", "3s"),
       ("TailscaleApiKey", ""),
       ("TailnetName", "");
