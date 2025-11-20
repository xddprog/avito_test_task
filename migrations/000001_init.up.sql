
CREATE TABLE IF NOT EXISTS teams (
    name VARCHAR(255) PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    team_name VARCHAR(255) NOT NULL,
    
    CONSTRAINT fk_team FOREIGN KEY (team_name) 
        REFERENCES teams(name) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pull_requests (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'OPEN', 
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    merged_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT fk_author FOREIGN KEY (author_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS pr_reviewers (
    pr_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,

    PRIMARY KEY (pr_id, user_id), 

    CONSTRAINT fk_pr FOREIGN KEY (pr_id) 
        REFERENCES pull_requests(id) ON DELETE CASCADE,
    CONSTRAINT fk_reviewer FOREIGN KEY (user_id) 
        REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_users_team ON users(team_name);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(is_active);
CREATE INDEX IF NOT EXISTS idx_pr_author ON pull_requests(author_id);