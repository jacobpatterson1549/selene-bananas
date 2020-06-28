CREATE OR REPLACE FUNCTION user_create
	( INOUT username VARCHAR
	, IN password CHAR
	) RETURNS SETOF VARCHAR
AS
$$
	INSERT
	INTO users
		( username
		, password
		)
	SELECT
		user_create.username
		, user_create.password
	ON CONFLICT (username) DO NOTHING
	RETURNING username
$$
LANGUAGE SQL;
