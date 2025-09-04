CREATE TABLE `web_product`
(
    `id`         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary key ID',
    `code`       VARCHAR(45)     NOT NULL DEFAULT '' COMMENT 'Business code',
    `name`       VARCHAR(255)    NOT NULL DEFAULT '' COMMENT 'Name',
    `mode`       TINYINT         NOT NULL DEFAULT 0 COMMENT 'Mode',
    `extra`      JSON            NOT NULL COMMENT 'Extra information',
    `version`    INT             NOT NULL DEFAULT 0 COMMENT 'Optimistic lock version',
    `created_at` TIMESTAMP(1)    NOT NULL DEFAULT CURRENT_TIMESTAMP(1) COMMENT 'Creation time',
    `updated_at` TIMESTAMP(1)    NULL     DEFAULT NULL ON UPDATE CURRENT_TIMESTAMP(1) COMMENT 'Update time',
    `deleted_at` TIMESTAMP(1)    NULL     DEFAULT NULL COMMENT 'Deletion time',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_code` (`code`)
) ENGINE = InnoDB
  AUTO_INCREMENT = 12119375
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='Web Product table';