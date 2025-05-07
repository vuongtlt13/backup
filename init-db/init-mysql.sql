-- Create test databases
CREATE DATABASE IF NOT EXISTS db1;
CREATE DATABASE IF NOT EXISTS db2;

-- Use db1
USE db1;

-- Create test table in db1
CREATE TABLE IF NOT EXISTS test_table (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data in db1
INSERT INTO test_table (name) VALUES
    ('Test Entry 1'),
    ('Test Entry 2'),
    ('Test Entry 3');

-- Use db2
USE db2;

-- Create test table in db2
CREATE TABLE IF NOT EXISTS test_table (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data in db2
INSERT INTO test_table (name) VALUES
    ('Test Entry 1'),
    ('Test Entry 2'),
    ('Test Entry 3'); 