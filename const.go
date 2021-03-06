package main

var separators = []string{
	"AND", "OR",
}

var operators = []string{
	"=", ">", "<", "!=", "<>", ">=", "<=",
}

var forbidden = []string{
	"ORDER BY", "LIMIT", "OFFSET", "HAVING", "GROUP BY", "SELECT", "FROM",
	"LEFT", "RIGHT", "FULL", "JOIN", "ON", "INSERT", "INTO", "VALUES",
}
