CREATE TABLE orders (
    id              VARCHAR(64)  NOT NULL PRIMARY KEY,
    user_id         VARCHAR(64)  NOT NULL,
    status          VARCHAR(32)  NOT NULL,
    amount_cents    BIGINT       NOT NULL,
    currency        VARCHAR(8)   NOT NULL,
    items_json      JSON         NOT NULL,
    idempotency_key VARCHAR(64)  DEFAULT NULL,
    version         INT          NOT NULL DEFAULT 0,
    created_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
