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
CREATE TABLE IF NOT EXISTS "parking_permits"
(
    "id"            UUID PRIMARY KEY,
    "permit_number" BIGINT                         NOT NULL,
    "created_by"    BIGINT                         NOT NULL,
    "updated_at"    TIMESTAMP(0) DEFAULT now(),
    "expires_at"    TIMESTAMP(0) NOT NULL
);

CREATE TABLE IF NOT EXISTS "complaints"
(
    "id"               UUID NOT NULL PRIMARY KEY,
    "created_by"       BIGINT               NOT NULL,
    "category"         "Complaint_Category" NOT NULL  DEFAULT "Complaint_Category" 'other',
    "title"            VARCHAR              NOT NULL,
    "description"      TEXT                 NOT NULL,
    "unit_number"      SMALLINT             NOT NULL,
    "status"           "Status"             NOT NULL  DEFAULT "Status" 'open',
    "updated_at"       TIMESTAMP(0)         DEFAULT now(),
    "created_at"       TIMESTAMP(0)         DEFAULT now()
);

CREATE TABLE IF NOT EXISTS "work_orders"
(
    "id"           UUID NOT NULL PRIMARY KEY,
    "created_by"   BIGINT          NOT NULL,
    "category"     "Work_Category" NOT NULL,
    "title"        VARCHAR         NOT NULL,
    "description"  TEXT            NOT NULL,
    "unit_number"  SMALLINT        NOT NULL,
    "status"       "Status"        NOT NULL       DEFAULT "Status" 'open',
    "updated_at"   TIMESTAMP(0) DEFAULT now(),
    "created_at"   TIMESTAMP(0) DEFAULT now()
);

CREATE TYPE "Account_Status" AS ENUM ('active', 'inactive', 'suspended');
CREATE TYPE "Role" AS ENUM ('tenant', 'admin');
CREATE TABLE IF NOT EXISTS "users"
(
    "id"          BI PRIMARY KEY,
    "clerk_id"    TEXT UNIQUE                    NOT NULL,
    "first_name"  VARCHAR                        NOT NULL,
    "last_name"   VARCHAR                        NOT NULL,
    "email"       VARCHAR                        NOT NULL,
    "phone"       VARCHAR                        NULL,
    "unit_number" SMALLINT                       NULL,
    "role"        "Role"                         NOT NULL DEFAULT "Role" 'tenant',
    "status"      "Account_Status"               NOT NULL DEFAULT "Account_Status" 'active',
    "last_login"  TIMESTAMP(0) NOT NULL,
    "updated_at"  TIMESTAMP(0)          DEFAULT now(),
    "created_at"  TIMESTAMP(0)          DEFAULT now()
);
CREATE INDEX "user_clerk_id_index" ON "users" ("clerk_id");
CREATE INDEX "user_unit_number_index" ON "users" ("unit_number");

CREATE TABLE IF NOT EXISTS "apartments"
(
    "id"               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "unit_number"      SMALLINT       NOT NULL,
    "price"            NUMERIC(10, 2) NOT NULL,
    "size"             SMALLINT       NOT NULL,
    "management_id"    BIGINT         NOT NULL,
    "availability"     BOOLEAN        NOT NULL        DEFAULT false,
    "lease_id"         BIGINT         NOT NULL,
    "lease_start_date" DATE           NOT NULL,
    "lease_end_date"   DATE           NOT NULL,
    "updated_at"       TIMESTAMP(0) DEFAULT now(),
    "created_at"       TIMESTAMP(0) DEFAULT now()
);
CREATE INDEX "apartment_unit_number_index" ON "apartments" ("unit_number");

CREATE TABLE IF NOT EXISTS "lockers"
(
    "id"          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    "access_code" varchar,
    "in_use"      BOOLEAN NOT NULL DEFAULT false,
    "user_id"     BIGINT
);
