CREATE TABLE `web_product_user_relation`
(
    `id`          BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Auto-increment primary key',
    `product_id`  BIGINT(20) UNSIGNED NOT NULL COMMENT 'Product ID',
    `user_id`     BIGINT(20) UNSIGNED NOT NULL COMMENT 'User ID',
    `create_date` TIMESTAMP(1)        NOT NULL DEFAULT CURRENT_TIMESTAMP(1) COMMENT 'Creation timestamp',
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_product_user` (`product_id`, `user_id`),
    KEY           `idx_user_id` (`user_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_unicode_ci COMMENT ='Product and user relation table';