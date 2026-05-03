CREATE TABLE IF NOT EXISTS metrics (
    id TEXT NOT NULL,
    type TEXT NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    CONSTRAINT metrics_pkey PRIMARY KEY (id, type),
    CONSTRAINT metrics_type_check CHECK (type IN ('counter', 'gauge')),
    CONSTRAINT metrics_value_check CHECK (
        (type = 'counter' AND delta IS NOT NULL AND value IS NULL)
        OR
        (type = 'gauge' AND value IS NOT NULL AND delta IS NULL)
    )
);
