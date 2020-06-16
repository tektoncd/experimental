.open results.db
CREATE TABLE taskrun (logid binary(16) PRIMARY KEY, taskrunlog BLOB, uid INTEGER, name TEXT, namespace TEXT);