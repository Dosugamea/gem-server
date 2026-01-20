-- Add requester column to transactions table
ALTER TABLE transactions 
ADD COLUMN requester VARCHAR(255) COMMENT 'リクエスト元（サービス名やユーザーIDなど）',
ADD INDEX idx_requester (requester);
