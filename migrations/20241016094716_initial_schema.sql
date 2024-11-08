-- +goose Up
-- +goose StatementBegin
-- Enable UUID extension

CREATE TABLE users (
                       telegram_id BIGINT PRIMARY KEY,
                       handle VARCHAR(255) UNIQUE,
                       username VARCHAR(255),
                       referrer_id BIGINT REFERENCES users(telegram_id),
                       referrals INTEGER DEFAULT 0,
                       points INTEGER DEFAULT 0,
                       join_waitlist BOOLEAN DEFAULT FALSE,
                       registration_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                       last_auth_date TIMESTAMP
);
CREATE INDEX idx_users_points ON users(points DESC);

CREATE TABLE quest_types (
                             id INTEGER PRIMARY KEY,
                             name VARCHAR(255) NOT NULL,
                             description TEXT
);

CREATE TABLE action_types (
                              id INTEGER PRIMARY KEY,
                              name VARCHAR(255) NOT NULL,
                              description TEXT
);

INSERT INTO quest_types (id, name, description) VALUES
                                                    (1, 'daily', 'Quests that reset daily'),
                                                    (2, 'weekly', 'Quests that reset weekly'),
                                                    (3, 'partnership', 'Partner promotional quests');

INSERT INTO action_types (id, name, description) VALUES
                                                     (1, 'follow', 'Follow action on social media'),
                                                     (2, 'website', 'Visit or interact with website');

CREATE TABLE social_quests (
                               quest_id UUID PRIMARY KEY,
                               quest_type_id INTEGER REFERENCES quest_types(id),
                               action_type_id INTEGER REFERENCES action_types(id),
                               image VARCHAR(255),
                               title VARCHAR(255),
                               description TEXT,
                               point_reward INTEGER,
                               created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                               available_from TIMESTAMP,
                               expires_at TIMESTAMP,
                               link VARCHAR(255),
                               chat_id BIGINT
);

CREATE TABLE users_social_quests (
                                     user_telegram_id BIGINT REFERENCES users(telegram_id),
                                     social_quest_id UUID REFERENCES social_quests(quest_id),
                                     completed BOOLEAN DEFAULT FALSE,
                                     started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                     finished_at TIMESTAMP,

                                     PRIMARY KEY (user_telegram_id, social_quest_id)
);

CREATE TABLE social_quest_validation_kinds (
                                               validation_id SERIAL PRIMARY KEY,
                                               validation_name VARCHAR(255) UNIQUE NOT NULL
);


CREATE TABLE user_validations (
                                  user_telegram_id BIGINT REFERENCES users(telegram_id),
                                  validation_id INTEGER REFERENCES social_quest_validation_kinds(validation_id),
                                  achieved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                                  PRIMARY KEY (user_telegram_id, validation_id)
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

CREATE TABLE referral_quests (
                                 quest_id UUID PRIMARY KEY,
                                 referrals_required INTEGER NOT NULL,
                                 point_reward INTEGER NOT NULL,
                                 created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE referral_quests_users (
                                       user_telegram_id BIGINT REFERENCES users(telegram_id),
                                       quest_id UUID REFERENCES referral_quests(quest_id),
                                       completed BOOLEAN DEFAULT FALSE,
                                       started_at TIMESTAMP,
                                       finished_at TIMESTAMP,
                                       PRIMARY KEY (user_telegram_id, quest_id)
);

CREATE INDEX idx_referral_quests_users_completed ON referral_quests_users(completed);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS daily_quests;
DROP TABLE IF EXISTS quest_validations;
DROP TABLE IF EXISTS user_validations;
DROP TABLE IF EXISTS social_quest_validation_kinds;
DROP TABLE IF EXISTS users_social_quests;
DROP TABLE IF EXISTS referral_quests_users;
DROP TABLE IF EXISTS social_quests;
DROP TABLE IF EXISTS referral_quests;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS action_types;
DROP TABLE IF EXISTS quest_types;
-- +goose StatementEnd
