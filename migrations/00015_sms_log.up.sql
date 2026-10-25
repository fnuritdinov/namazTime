CREATE TABLE sms_log
(
    id          BIGSERIAL PRIMARY KEY,
    device_id   TEXT        NOT NULL,
    phone_hash  TEXT        NOT NULL,
    provider_id TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX sms_log_device_idx ON sms_log (device_id, created_at);
CREATE INDEX sms_log_phone_idx ON sms_log (phone_hash, created_at);
CREATE INDEX sms_log_created_idx ON sms_log (created_at);