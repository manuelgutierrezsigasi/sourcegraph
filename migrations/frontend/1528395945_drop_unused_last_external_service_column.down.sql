BEGIN;

ALTER TABLE IF EXISTS gitserver_repos ADD COLUMN last_external_service bigint;

COMMIT;
