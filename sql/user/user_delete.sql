CREATE OR REPLACE FUNCTION user_delete
	( INOUT username VARCHAR
	) RETURNS SETOF VARCHAR
AS
$$
	DELETE
	FROM users
	AS u
	WHERE u.username = user_delete.username
	RETURNING u.username
$$
LANGUAGE SQL;
