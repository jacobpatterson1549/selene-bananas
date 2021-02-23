package main

import "embed"

//go:embed embed/version.txt
var embedVersion string

//go:embed embed/sql/users.sql
//go:embed embed/sql/user_create.sql
//go:embed embed/sql/user_read.sql
//go:embed embed/sql/user_update_password.sql
//go:embed embed/sql/user_update_points_increment.sql
//go:embed embed/sql/user_delete.sql
var embeddedSQLFS embed.FS
