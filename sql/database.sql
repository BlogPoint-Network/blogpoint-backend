CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(10) UNIQUE NOT NULL,
    description TEXT
);

INSERT INTO roles (name, description) VALUES
    ('user', 'Standard user role'),
    ('moderator', 'Moderator role');

CREATE TABLE files (
    id SERIAL PRIMARY KEY,
    filename varchar(200) UNIQUE NOT NULL,
    mime_type varchar(30) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- CREATE TABLE files (
--     id SERIAL PRIMARY KEY,
--     owner_id INT NOT NULL,
--     filename varchar(200) UNIQUE NOT NULL,
--     url VARCHAR(200) NOT NULL,
--     mime_type varchar(30) NOT NULL,
--     used_in VARCHAR(20) NOT NULL CHECK (used_in IN ('user_avatar', 'channel_avatar', 'post_preview', 'post_content', 'post_media')),
--     entity_id INT NOT NULL,
--     name VARCHAR(100) DEFAULT NULL,
--     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
-- );

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    role_id INT REFERENCES roles(id) DEFAULT 1,
    email VARCHAR(100) UNIQUE NOT NULL,
    login VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    logo_id INT REFERENCES files(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE files ADD COLUMN user_id INT;
ALTER TABLE files ADD CONSTRAINT fk_files_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

CREATE TABLE verification_codes (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code VARCHAR(6) NOT NULL,
    type TEXT CHECK (type IN ('email_verification', 'account_deletion', 'password_reset')),
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_id, type)
);

CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    color VARCHAR(7) NOT NULL
);

CREATE TABLE tags (
    id SERIAL PRIMARY KEY,
    category_id INT REFERENCES categories(id) ON DELETE CASCADE,
    name VARCHAR(100) UNIQUE NOT NULL,
    color VARCHAR(7) NOT NULL
);

CREATE TABLE channels (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT DEFAULT '',
    category_id INT REFERENCES categories(id) ON DELETE SET NULL,
    owner_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subs_count INT NOT NULL DEFAULT 0 CHECK (subs_count >= 0),
    logo_id INT REFERENCES files(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE subscriptions (
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    channel_id INT REFERENCES channels(id) ON DELETE CASCADE,
    signed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, channel_id)
);

CREATE TABLE channel_moderators (
    channel_id INT REFERENCES channels(id) ON DELETE CASCADE,
    moderator_id INT REFERENCES users(id) ON DELETE CASCADE,
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, moderator_id)
);

CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    channel_id INT REFERENCES channels(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    likes_count INT DEFAULT 0,
    dislikes_count INT DEFAULT 0,
    views_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE post_tags (
    post_id INT REFERENCES posts(id) ON DELETE CASCADE,
    tag_id INT REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

CREATE TABLE post_reactions (
    id SERIAL PRIMARY KEY,
    post_id INT REFERENCES posts(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    reaction BOOLEAN NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_reaction_per_user UNIQUE (user_id, post_id)
);

CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    post_id INT REFERENCES posts(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_deleted BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE comments ADD COLUMN parent_id INT REFERENCES comments(id) ON DELETE CASCADE;

CREATE TYPE target AS ENUM ('channel', 'post', 'comment');
CREATE TYPE status AS ENUM ('open', 'in progress', 'close');

CREATE TABLE complaints (
    id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    target_type target NOT NULL,
    target_id INT NOT NULL,
    complaint_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    status status DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);


CREATE OR REPLACE FUNCTION update_likes_dislikes() RETURNS TRIGGER AS $$
BEGIN
    CASE TG_OP
        WHEN 'INSERT' THEN
            UPDATE posts
            SET likes_count = likes_count + (NEW.reaction::INT),
                dislikes_count = dislikes_count + ((NOT NEW.reaction)::INT)
            WHERE id = NEW.post_id;

        WHEN 'DELETE' THEN
            UPDATE posts
            SET likes_count = likes_count - (OLD.reaction::INT),
                dislikes_count = dislikes_count - ((NOT OLD.reaction)::INT)
            WHERE id = OLD.post_id;

        WHEN 'UPDATE' THEN
            UPDATE posts
            SET likes_count = likes_count + (NEW.reaction::INT) - (OLD.reaction::INT),
                dislikes_count = dislikes_count + ((NOT NEW.reaction)::INT) - ((NOT OLD.reaction)::INT)
            WHERE id = NEW.post_id;
    END CASE;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_likes_dislikes_trigger
AFTER INSERT OR DELETE OR UPDATE ON post_reactions
FOR EACH ROW EXECUTE FUNCTION update_likes_dislikes();
