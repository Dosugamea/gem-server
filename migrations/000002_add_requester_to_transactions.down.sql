-- Remove requester column from transactions table
ALTER TABLE transactions 
DROP INDEX idx_requester,
DROP COLUMN requester;
