DROP INDEX IF EXISTS customers_email_account_unique;
CREATE UNIQUE INDEX customer_email_account_unique ON customer(email,has_account) WHERE deleted_at IS NULL AND email IS NOT NULL;
