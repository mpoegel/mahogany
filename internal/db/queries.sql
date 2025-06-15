-- name: ListDevices :many
SELECT * FROM devices
ORDER BY hostname;

-- name: AddDevice :one
INSERT INTO devices (
  hostname
) VALUES (
  ?
)
RETURNING *;

-- name: GetDevice :one
SELECT * FROM devices
WHERE hostname = ?;

-- name: UpdateDevice :exec
UPDATE devices
SET tailscale_last_seen = ?,
    agent_last_seen = ?
WHERE hostname = ?;

-- name: CountDevices :one
SELECT COUNT(*) FROM devices;

-- name: DeleteDevice :exec
DELETE FROM devices
WHERE id = ?;

-- name: ListPackages :many
SELECT * FROM packages
ORDER BY name;

-- name: AddPackage :one
INSERT INTO packages (
  name, install_cmd, update_cmd, remove_cmd
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: UpdatePackage :exec
UPDATE packages
set name = ?,
    install_cmd = ?,
    update_cmd = ?,
    remove_cmd = ?
WHERE id = ?;

-- name: DeletePackage :exec
DELETE FROM packages
WHERE id = ?;

-- name: ListAssets :many
SELECT * FROM assets
ORDER BY id;

-- name: ListAssetsOnDevice :many
SELECT * FROM assets
WHERE device_id = ?
ORDER BY id;

-- name: ListAssetsForPackage :many
SELECT * FROM assets
WHERE package_id = ?
ORDER BY id;

-- name: AddAsset :one
INSERT INTO assets (
  device_id, package_id, source_url, version
) VALUES (
  ?, ?, ?, ?
)
RETURNING *;

-- name: GetSetting :one
SELECT name, value FROM settings
WHERE name = ?
LIMIT 1;

-- name: UpdateSetting :exec
UPDATE settings
set value = ?
WHERE name = ?;

-- name: ListSettings :many
SELECT * FROM settings
ORDER BY id;

-- name: ListWatchedServices :many
SELECT name FROM watched_services ORDER BY name;

-- name: AddWatchedService :exec
INSERT INTO watched_services (name) VALUES (?);

-- name: DeleteWatchedService :exec
DELETE FROM watched_services WHERE name = ?;

-- name: ListTrackedServices :many
SELECT * FROM tracked_services ORDER BY (device_id, name);

-- name: GetTrackedServiceID :one
SELECT id FROM tracked_services WHERE name=? and device_id=?;

-- name: AddTrackedService :one
INSERT INTO tracked_services (
  device_id, name, status, last_updated, container_id, container_image
) VALUES (
  ?, ?, ?, ?, ?, ?
)
RETURNING *;

-- name: DeleteTrackedService :exec
DELETE FROM tracked_services WHERE id = ?;

-- name: UpdateTrackedService :exec
UPDATE tracked_services
set status = ?,
    last_updated = ?
WHERE id = ?;
