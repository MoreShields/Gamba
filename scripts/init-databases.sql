-- Dev database setupC
-- Create separate databases for each service
CREATE DATABASE gambler_db;
CREATE DATABASE lol_tracker_db;

-- Grant all privileges to the gambler user
GRANT ALL PRIVILEGES ON DATABASE gambler_db TO gambler;
GRANT ALL PRIVILEGES ON DATABASE lol_tracker_db TO gambler;