-- results_id serve as auto-generated key by sqltite to uniquely identify a taskrun log
CREATE TABLE taskrun (
	results_id binary(16) PRIMARY KEY,
	taskrunlog BLOB,
	name TEXT,
	namespace TEXT
);
