-- +goose Up
-- create enum type "order_status"
CREATE TYPE "order_status" AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');
-- create "users" table
CREATE TABLE "users" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "username" character varying(255) NOT NULL,
  "password" text NOT NULL,
  "balance" numeric(10,2) NULL DEFAULT 0.00,
  "withdrawn" numeric(10,2) NULL DEFAULT 0.00,
  PRIMARY KEY ("id"),
  CONSTRAINT "users_username_key" UNIQUE ("username")
);
-- create "orders" table
CREATE TABLE "orders" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NULL,
  "order_num" character varying(255) NOT NULL,
  "status" "order_status" NOT NULL DEFAULT 'NEW',
  "amount" numeric(10,2) NULL DEFAULT NULL::numeric,
  "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  "updated_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "orders_order_num_key" UNIQUE ("order_num"),
  CONSTRAINT "orders_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- create "withdrawals" table
CREATE TABLE "withdrawals" (
  "id" uuid NOT NULL DEFAULT gen_random_uuid(),
  "user_id" uuid NULL,
  "amount" numeric(10,2) NOT NULL,
  "created_at" timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY ("id"),
  CONSTRAINT "withdrawals_user_id_fkey" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);

-- +goose Down
-- reverse: create "withdrawals" table
DROP TABLE "withdrawals";
-- reverse: create "orders" table
DROP TABLE "orders";
-- reverse: create "users" table
DROP TABLE "users";
-- reverse: create enum type "order_status"
DROP TYPE "order_status";
