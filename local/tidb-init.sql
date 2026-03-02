-- Schema initialisation for container-build-service (local development).
-- Run automatically by the tidb-init service in docker-compose.yaml.

CREATE DATABASE IF NOT EXISTS buildservice;

USE buildservice;

CREATE TABLE IF NOT EXISTS project_versions (
  project    VARCHAR(255) NOT NULL PRIMARY KEY,
  version    VARCHAR(32)  NOT NULL DEFAULT '0.1.0',
  updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS build_state (
  repo               VARCHAR(255) NOT NULL PRIMARY KEY,
  last_processed_sha CHAR(40)     NOT NULL,
  updated_at         TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS build_records (
  id         BIGINT       NOT NULL AUTO_INCREMENT PRIMARY KEY,
  project    VARCHAR(255) NOT NULL,
  commit_sha CHAR(40)     NOT NULL,
  status     ENUM('pending','success','failure') NOT NULL DEFAULT 'pending',
  claimed_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_project_sha (project, commit_sha)
);
