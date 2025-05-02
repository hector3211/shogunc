CREATE TYPE "Complaint_Category" AS ENUM (
    'maintenance',
    'noise',
    'security',
    'parking',
    'neighbor',
    'trash',
    'internet',
    'lease',
    'natural_disaster',
    'other'
);
CREATE TYPE "Status" AS ENUM (
    'open',
    'in_progress',
    'resolved',
    'closed'
);
CREATE TYPE "Type" AS ENUM (
    'lease_agreement',
    'amendment',
    'extension',
    'termination',
    'addendum'
);
CREATE TYPE "Lease_Status" AS ENUM (
    'draft',
    'pending_approval',
    'active',
    'expired',
    'terminated',
    'renewed'
);
CREATE TYPE "Compliance_Status" AS ENUM (
    'pending_review',
    'compliant',
    'non_compliant',
    'exempted'
);
CREATE TYPE "Work_Category" AS ENUM (
    'plumbing',
    'electric',
    'carpentry',
    'hvac',
    'other'
);

CREATE TABLE IF NOT EXISTS "parking_permits" (
    "id"            UUID PRIMARY KEY,
    "permit_number" BIGINT NOT NULL,
    "created_by"    BIGINT NOT NULL,
    "updated_at"    TIMESTAMP DEFAULT now(),
    "expires_at"    TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS "lockers" (
    "id"          UUID PRIMARY KEY,
    "access_code" VARCHAR,
    "in_use"      BOOLEAN NOT NULL DEFAULT false,
    "user_id"     BIGINT
);
