CREATE OR REPLACE FUNCTION user_read
	( IN username VARCHAR
	) RETURNS SETOF users
AS
$$
	SELECT u.username
		, u.password
		, u.points
	FROM users
	AS u
	WHERE u.username = user_read.username
$$
LANGUAGE SQL;
