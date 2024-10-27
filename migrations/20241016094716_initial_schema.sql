-- +goose Up
-- +goose StatementBegin
-- Enable UUID extension

-- Create users table
CREATE TABLE users (
                       telegram_id BIGINT PRIMARY KEY,
                       handle VARCHAR(255) UNIQUE,
                       username VARCHAR(255),
                       referrer_id BIGINT REFERENCES users(telegram_id),
                       referrals INTEGER DEFAULT 0,
                       points INTEGER DEFAULT 0,
                       profile_image VARCHAR(255),
                       join_waitlist BOOLEAN DEFAULT FALSE,
                       registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       last_auth_date TIMESTAMP
);
CREATE INDEX idx_users_points ON users(points DESC);

CREATE TABLE social_quests (
                               quest_id UUID PRIMARY KEY,
                               image VARCHAR(255),
                               title VARCHAR(255),
                               description TEXT,
                               point_reward INTEGER,
                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users_social_quests (
                                     user_telegram_id BIGINT REFERENCES users(telegram_id),
                                     social_quest_id UUID REFERENCES social_quests(quest_id),
                                     completed BOOLEAN DEFAULT FALSE,
                                     started_at TIMESTAMP,
                                     finished_at TIMESTAMP,
                                     PRIMARY KEY (user_telegram_id, social_quest_id)
);

CREATE TABLE social_quest_validation_kinds (
                                               validation_id SERIAL PRIMARY KEY,
                                               validation_name VARCHAR(255) UNIQUE
);

CREATE TABLE quest_validations (
                                   quest_id UUID REFERENCES social_quests(quest_id),
                                   validation_id INTEGER REFERENCES social_quest_validation_kinds(validation_id),
                                   PRIMARY KEY (quest_id, validation_id)
);

CREATE TABLE daily_quests (
                              user_telegram_id BIGINT PRIMARY KEY REFERENCES users(telegram_id),
                              last_claimed_at TIMESTAMP,
                              consecutive_days_claimed INTEGER DEFAULT 0
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS daily_quests;
DROP TABLE IF EXISTS quest_validations;
DROP TABLE IF EXISTS social_quest_validation_kinds;
DROP TABLE IF EXISTS users_social_quests;
DROP TABLE IF EXISTS social_quests;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "uuid-ossp";
-- +goose StatementEnd
