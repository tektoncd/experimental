
CREATE TABLE results (
	parent varchar(64),
	id varchar(64),

	name varchar(64),
	data BLOB,

	PRIMARY KEY(parent, id)
);
CREATE UNIQUE INDEX resultsByName ON results(parent, name);

CREATE TABLE records (
	parent varchar(64),
	result_id varchar(64),
	id varchar(64),

	result_name varchar(64),
	name varchar(64),
	data BLOB,

	PRIMARY KEY(parent, result_id, id)
);
CREATE UNIQUE INDEX recordsByName ON records(parent, result_name, name);