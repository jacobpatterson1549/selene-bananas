CREATE OR REPLACE FUNCTION user_update_points_increment
	( INOUT username VARCHAR
	, IN points_delta INT
	) RETURNS SETOF VARCHAR
AS
$$
	UPDATE users
	AS u
	SET points = points + user_update_points_increment.points_delta
	WHERE u.username = user_update_points_increment.username
	RETURNING u.username
$$
LANGUAGE SQL;
