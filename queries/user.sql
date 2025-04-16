-- name: CreateUser :one
INSERT INTO users (
    clerk_id,
    first_name,
    last_name,
    email,
    phone,
    role,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now()
) RETURNING id, clerk_id, first_name, last_name, email, phone, role, created_at;

-- name: GetUser :one
SELECT id, first_name, last_name, email, phone, role, created_at
FROM users
WHERE id = $1;

-- name: GetUserByClerkId :one
SELECT id, clerk_id, first_name, last_name, email, phone, role, status, created_at
FROM users
WHERE first_name = 'hector'
LIMIT 1;

-- name: GetUserByClerkIdTwo :one
SELECT id, clerk_id, first_name, created_at
FROM users
WHERE clerk_id = $1 AND first_name = 'hector'
LIMIT 1 OFFSET 20;
