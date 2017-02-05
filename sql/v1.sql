CREATE SCHEMA IF NOT EXISTS data;

SET SCHEMA 'data';

CREATE TABLE IF NOT EXISTS chat_user (
    id serial PRIMARY KEY,
    user_name varchar(64) UNIQUE,
    hashed_password varchar(1024)
);

CREATE TABLE IF NOT EXISTS chat_room (
    id serial PRIMARY KEY,
    room_name varchar(1024) UNIQUE,
    created_by varchar(64)
);

CREATE TABLE IF NOT EXISTS chat_member (
    chat_user_id integer REFERENCES chat_user,
    chat_room_id integer REFERENCES chat_room
);

CREATE TABLE IF NOT EXISTS chat_message (
    id serial PRIMARY KEY,
    time_sent TIMESTAMP,
    sent_by varchar(64),
    chat_room_id integer REFERENCES chat_room,
    contents varchar(4096)
);
