CREATE TABLE `user`
(
    `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    `code`         VARCHAR(45)     NOT NULL DEFAULT '' COMMENT 'Business code',
    `name`         VARCHAR(255)    NOT NULL DEFAULT '' COMMENT 'User name (advertiser or company name)',
    `type`         TINYINT         NOT NULL DEFAULT 0 COMMENT 'User type: 0=individual, 1=enterprise, 2=agency',
    `region`       VARCHAR(100)    NOT NULL DEFAULT '' COMMENT 'User region/location',
    `email`        VARCHAR(255)    NOT NULL DEFAULT '' COMMENT 'Contact email',
    `phone`        VARCHAR(50)     NOT NULL DEFAULT '' COMMENT 'Contact phone',
    `status`       TINYINT         NOT NULL DEFAULT 1 COMMENT 'User status: 0=inactive, 1=active, 2=suspended',
    `extra`        JSON            NOT NULL COMMENT 'Extra information',
    `version`      INT             NOT NULL DEFAULT 0 COMMENT 'Optimistic lock version',
    `created_at`   TIMESTAMP(1)    NOT NULL DEFAULT CURRENT_TIMESTAMP(1) COMMENT 'Creation time',
    `updated_at`   TIMESTAMP(1)    NULL     DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(1) COMMENT 'Update time',
    `deleted_at`   TIMESTAMP(1)    NULL     DEFAULT NULL COMMENT 'Deletion time',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`),
    UNIQUE KEY `uk_email` (`email`),
    KEY `idx_region_type` (`region`, `type`),
    KEY `idx_status` (`status`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 10000
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='User table';