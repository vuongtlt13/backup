-- Create test databases
CREATE DATABASE db1;
CREATE DATABASE db2;

-- Connect to db1
\c db1;

-- Create test table in db1
CREATE TABLE IF NOT EXISTS test_table (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data in db1
INSERT INTO test_table (name) VALUES
    ('Test Entry 1'),
    ('Test Entry 2'),
    ('Test Entry 3');

-- Connect to db2
\c db2;

-- Create test table in db2
CREATE TABLE IF NOT EXISTS test_table (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data in db2
INSERT INTO test_table (name) VALUES
    ('Test Entry 1'),
    ('Test Entry 2'),
    ('Test Entry 3'); 