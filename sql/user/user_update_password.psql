CREATE OR REPLACE FUNCTION user_update_password
	( INOUT username VARCHAR
	, IN password CHAR
	) RETURNS SETOF VARCHAR
AS
$$
	UPDATE users
	AS u
	SET password = user_update_password.password
	WHERE u.username = user_update_password.username
	RETURNING u.username
$$
LANGUAGE SQL;
