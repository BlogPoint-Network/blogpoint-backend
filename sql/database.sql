CREATE TABLE roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(10) UNIQUE NOT NULL,
    description TEXT
);

INSERT INTO roles (name, description) VALUES
                                          ('user', 'Standard user role'),
                                          ('moderator', 'Moderator role');

CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    role_id INT REFERENCES roles(id) DEFAULT 1,
    email VARCHAR(100) UNIQUE NOT NULL,
    login VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE channels (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    owner_id INT REFERENCES users(id) ON DELETE CASCADE,
    subs_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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

CREATE TABLE blogs (
    id SERIAL PRIMARY KEY,
    channel_id INT REFERENCES channels(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    likes_count INT DEFAULT 0,
    dislikes_count INT DEFAULT 0,
    views_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE media_type AS ENUM ('image', 'video', 'audio');

CREATE TABLE blog_media (
    id SERIAL PRIMARY KEY,
    blog_id INT REFERENCES blogs(id) ON DELETE CASCADE,
    media_type media_type NOT NULL,
    media_url TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE reaction AS ENUM ('like', 'dislike');

CREATE TABLE blog_reactions (
    id SERIAL PRIMARY KEY,
    blog_id INT REFERENCES blogs(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    reaction_type reaction NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_reaction_per_user UNIQUE (user_id, blog_id)
);

CREATE TABLE comments (
    id SERIAL PRIMARY KEY,
    blog_id INT REFERENCES blogs(id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TYPE target AS ENUM ('channel', 'blog', 'comment');
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
    IF (TG_OP = 'INSERT') THEN
        IF (NEW.reaction_type = 'like') THEN
UPDATE blogs SET likes_count = likes_count + 1 WHERE id = NEW.blog_id;
ELSIF (NEW.reaction_type = 'dislike') THEN
UPDATE blogs SET dislikes_count = dislikes_count + 1 WHERE id = NEW.blog_id;
END IF;
    ELSIF (TG_OP = 'DELETE') THEN
        IF (OLD.reaction_type = 'like') THEN
UPDATE blogs SET likes_count = likes_count - 1 WHERE id = OLD.blog_id;
ELSIF (OLD.reaction_type = 'dislike') THEN
UPDATE blogs SET dislikes_count = dislikes_count - 1 WHERE id = OLD.blog_id;
END IF;
    ELSIF (TG_OP = 'UPDATE') THEN
        IF (OLD.reaction_type = 'like' AND NEW.reaction_type = 'dislike') THEN
UPDATE blogs SET likes_count = likes_count - 1, dislikes_count = dislikes_count + 1 WHERE id = NEW.blog_id;
ELSIF (OLD.reaction_type = 'dislike' AND NEW.reaction_type = 'like') THEN
UPDATE blogs SET likes_count = likes_count + 1, dislikes_count = dislikes_count - 1 WHERE id = NEW.blog_id;
END IF;
END IF;
RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_likes_dislikes_trigger
    AFTER INSERT OR DELETE OR UPDATE ON blog_reactions
FOR EACH ROW EXECUTE FUNCTION update_likes_dislikes();