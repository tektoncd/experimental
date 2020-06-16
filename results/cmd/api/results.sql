-- logid serve as auto-generated key by sqltite to uniquely identify a taskrun log
-- uid is the key from taskrun itself we now use to do query on taskrun logs, future we may use cluster-info as key
.open results.db
CREATE TABLE taskrun (logid binary(16) PRIMARY KEY, taskrunlog BLOB, uid TEXT, name TEXT, namespace TEXT);
