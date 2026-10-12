
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    full_name VARCHAR(200) NOT NULL,
    email VARCHAR(200) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS candidates (
    id BIGSERIAL PRIMARY KEY,
    full_name VARCHAR(200) NOT NULL,
    age INTEGER NOT NULL CHECK (age > 0),
    gender VARCHAR(32) NOT NULL,
    email VARCHAR(200) NOT NULL,
    phone VARCHAR(64) NOT NULL,
    city VARCHAR(120) NOT NULL,
    position VARCHAR(160) NOT NULL,
    experience_years NUMERIC(4,1) NOT NULL DEFAULT 0,
    education VARCHAR(160) NOT NULL,
    employment_type VARCHAR(64) NOT NULL,
    salary_expectation INTEGER NOT NULL DEFAULT 0,
    about TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS vacancies (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    department VARCHAR(160),
    city VARCHAR(120) NOT NULL,
    employment_type VARCHAR(64) NOT NULL,
    min_experience NUMERIC(4,1) NOT NULL DEFAULT 0,
    education_level VARCHAR(160),
    salary_min INTEGER NOT NULL DEFAULT 0,
    salary_max INTEGER NOT NULL DEFAULT 0,
    description TEXT,
    requirements TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS skills (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS candidate_skills (
    candidate_id BIGINT NOT NULL REFERENCES candidates(id) ON DELETE CASCADE,
    skill_id BIGINT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (candidate_id, skill_id)
);

CREATE TABLE IF NOT EXISTS vacancy_skills (
    vacancy_id BIGINT NOT NULL REFERENCES vacancies(id) ON DELETE CASCADE,
    skill_id BIGINT NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    PRIMARY KEY (vacancy_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_candidates_city ON candidates(city);
CREATE INDEX IF NOT EXISTS idx_candidates_position ON candidates(position);
CREATE INDEX IF NOT EXISTS idx_candidates_experience ON candidates(experience_years);
CREATE INDEX IF NOT EXISTS idx_candidates_salary ON candidates(salary_expectation);
CREATE INDEX IF NOT EXISTS idx_vacancies_active ON vacancies(is_active);
