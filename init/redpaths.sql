CREATE TABLE redpaths_modules (
                               module_id INT GENERATED ALWAYS AS IDENTITY,
                               key VARCHAR(100) NOT NULL,
                               name VARCHAR(100) NOT NULL,
                               version VARCHAR(10) NOT NULL,
                               author VARCHAR(255) NOT NULL,
                               description VARCHAR(255) NOT NULL,
                               attack_id VARCHAR(255) NOT NULL,
                               loot_path VARCHAR(255) NOT NULL,
                               module_type VARCHAR(100) NOT NULL,
                               execution_metric VARCHAR(100) NOT NULL,
                               PRIMARY KEY(module_id),
                               UNIQUE(key)
);

CREATE TABLE redpaths_modules_dependencies (
                                     previous_module VARCHAR(100) NOT NULL,
                                     next_module VARCHAR(100) NOT NULL,
                                     PRIMARY KEY (previous_module, next_module),
                                     FOREIGN KEY (previous_module) REFERENCES redpaths_modules (key),
                                     FOREIGN KEY (next_module) REFERENCES redpaths_modules (key)
);


CREATE TABLE redpaths_modules_metadata (
    project_uid VARCHAR(255),
    module_id INT,
    CONSTRAINT fk_module
     FOREIGN KEY (module_id)
        REFERENCES redpaths_modules (module_id)
);


CREATE TABLE redpaths_users (
    user_id INT GENERATED ALWAYS AS IDENTITY,
    hash varchar,
    PRIMARY KEY (user_id)
);

CREATE TABLE redpaths_collections
(
    id INT GENERATED ALWAYS AS IDENTITY,
    name VARCHAR,
    description VARCHAR,
    PRIMARY KEY (id)
);

CREATE TABLE redpaths_collection_modules
(
    module_key    VARCHAR,
    collection_id INT,
    PRIMARY KEY (module_key, collection_id)
);

CREATE TABLE redpaths_modules_options
(
    module_key VARCHAR,
    option_key VARCHAR,
    label VARCHAR,
    placeholder VARCHAR,
    type VARCHAR,
    required bool,
    PRIMARY KEY (module_key, option_key)
);

CREATE TABLE redpaths_modules_runs
(
    module_key VARCHAR,
    run_uid VARCHAR,
    vector_run_uid VARCHAR,
    ran_at TIMESTAMP,
    project_uid VARCHAR,
    was_successful BOOLEAN,
    targets jsonb,
    parameters jsonb
);

CREATE TABLE redpaths_module_last_runs
(
    module_key VARCHAR,
    run_uid VARCHAR,
    ran_at TIMESTAMP,
    project_uid VARCHAR,
    parameter jsonb
);

CREATE TABLE redpaths_vector_runs
(
    run_uid VARCHAR,
    ran_at TIMESTAMP,
    project_uid VARCHAR,
    graph jsonb
);

CREATE TABLE redpaths_module_logs (
    id INT GENERATED ALWAYS AS IDENTITY,
    project_uid VARCHAR(255),
    module_key VARCHAR,
    run_uid VARCHAR,
    log_level VARCHAR,
    event_type VARCHAR,
    message VARCHAR,
    payload jsonb,
    timestamp TIMESTAMP,
    PRIMARY KEY (id)
);

CREATE TABLE redpaths_change_event
(
    event_id               INT GENERATED ALWAYS AS IDENTITY,
    project_uid         VARCHAR,
    -- Dgraph UID from node or edge
    subject_uid            VARCHAR NOT NULL,
    event_type             TEXT    NOT NULL
        CHECK (event_type IN (
                              'node_create',
                              'node_delete',
                              'attribute_update',
                              'edge_add',
                              'edge_remove'
            )),
    predicate              VARCHAR,
    -- on attribute events
    old_value              JSONB,
    new_value              JSONB,
    -- on edge events
    old_target_uid         VARCHAR,
    new_target_uid         VARCHAR,
    -- meta
    changed_at             TIMESTAMP,
    detected_by_module_key VARCHAR,
    module_run_uid         VARCHAR,
    transaction_uid        VARCHAR,
    PRIMARY KEY (event_id)
);

CREATE TABLE redpaths_node_snapshot (
                               node_uid   TEXT       PRIMARY KEY,
                               data       JSONB      NOT NULL,
                               edges      JSONB      NOT NULL DEFAULT '[]',
                               version    BIGINT     NOT NULL DEFAULT 0,
                               updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
