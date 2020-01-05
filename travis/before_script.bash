#!/usr/bin/env bash
set -eux

if [ "${PGVERSION-}" != "" ]
then
  psql -U postgres -d postgres -c "create user iidy password 'iidy'"
  psql -U postgres -d postgres -c "create database iidy with owner iidy"
  psql -U iidy -d iidy -c "create table lists (
	list     text    not null,
	item     text    not null,
	attempts integer not null default 0,
	constraint list_pk primary key (list, item))"
fi
