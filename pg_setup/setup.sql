-- TO RUN, DO THIS:
-- $ psql -X -U postgres -d postgres -f setup.sql 
create user iidy password 'iidy';
create database iidy with owner iidy;
\connect iidy
set role iidy;
create table lists (
	list     text    not null,
	item     text    not null,
	attempts integer not null default 0,
	constraint list_pk primary key (list, item));
