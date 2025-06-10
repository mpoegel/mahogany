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
