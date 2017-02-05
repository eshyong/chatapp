CREATE SCHEMA IF NOT EXISTS data;

SET SCHEMA 'data';

CREATE TABLE IF NOT EXISTS chat_users (
    id serial PRIMARY KEY,
    username varchar(64) UNIQUE,
    hashed_password varchar(1024)
);

CREATE TABLE IF NOT EXISTS chat_rooms (
    id serial PRIMARY KEY,
    name varchar(1024),
    created_by integer REFERENCES chat_users,
    UNIQUE (name, created_by)
);

CREATE TABLE IF NOT EXISTS chat_members (
    member_id integer REFERENCES chat_users,
    room_id integer REFERENCES chat_rooms
);

CREATE TABLE IF NOT EXISTS chat_messages (
    id serial PRIMARY KEY,
    time_sent TIMESTAMP,
    sent_by integer REFERENCES chat_users,
    posted_in integer REFERENCES chat_rooms,
    contents varchar(4096)
);
