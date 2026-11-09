-- +goose Up
-- An optional category per row, and the one a product last had, for suggestions; '' when none.
ALTER TABLE stock_item ADD COLUMN category TEXT NOT NULL DEFAULT '';
ALTER TABLE stock_product ADD COLUMN category TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE stock_product DROP COLUMN category;
ALTER TABLE stock_item DROP COLUMN category;
