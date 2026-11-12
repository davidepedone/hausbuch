-- +goose Up
-- The unit a product last had, for suggestions; '' when none.
ALTER TABLE stock_product ADD COLUMN unit TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE stock_product DROP COLUMN unit;
