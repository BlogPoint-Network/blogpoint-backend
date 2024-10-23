CREATE TABLE roles (
    role_id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT
);

CREATE TABLE users (
    user_id SERIAL PRIMARY KEY,
    role_id INT REFERENCES roles(role_id),
    email VARCHAR(100) UNIQUE NOT NULL,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE channels (
    channel_id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    owner_id INT REFERENCES users(user_id),
    subs_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
    user_id INT REFERENCES users(user_id),
    channel_id INT REFERENCES channels(channel_id),
    signed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, channel_id)
);

CREATE TABLE channel_moderators (
    channel_id INT REFERENCES channels(channel_id),
    moderator_id INT REFERENCES users(user_id),
    assigned_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (channel_id, moderator_id)
);

CREATE TABLE blogs (
    blog_id SERIAL PRIMARY KEY,
    channel_id INT REFERENCES channels(channel_id),
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    likes_count INT DEFAULT 0,
    dislikes_count INT DEFAULT 0,
    views_count INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE blog_media (
    media_id SERIAL PRIMARY KEY,
    blog_id INT REFERENCES blogs(blog_id),
    media_type VARCHAR(50) NOT NULL,
    media_url TEXT UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE blog_reactions (
    reaction_id SERIAL PRIMARY KEY,
    blog_id INT REFERENCES blogs(blog_id),
    user_id INT REFERENCES users(user_id),
    reaction_type VARCHAR(10) CHECK (reaction_type IN ('like', 'dislike')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE comments (
    comment_id SERIAL PRIMARY KEY,
    blog_id INT REFERENCES blogs(blog_id),
    user_id INT REFERENCES users(user_id),
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE complaints (
    complaint_id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(user_id),
    target_type VARCHAR(20) CHECK (target_type IN ('channel', 'blog', 'comment')) NOT NULL,
    target_id INT NOT NULL,
    complaint_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    status VARCHAR(20) DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
