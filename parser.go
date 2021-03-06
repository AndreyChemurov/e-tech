package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Masterminds/squirrel"
)

// Функция нахождения элемента в слайсе
func find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

func parseField(query string, pos int, statements map[string][]string) (map[string][]string, int, error) {
	var (
		field []byte = make([]byte, 0, 15)
	)

	for {
		if pos == len(query) {
			return nil, 0, errors.New("Expected EOL (;), got nothing")
		}

		if query[pos] == ';' {
			return nil, 0, fmt.Errorf("Unexpected EOL (;) at %d", pos)
		}

		if query[pos] == '.' && len(field) == 0 {
			return nil, 0, errors.New("Expected column, got dot")
		}

		if query[pos] == ' ' && len(field) != 0 {
			statements["names"] = append(statements["names"], string(field))

			pos++

			break
		}

		field = append(field, query[pos])
		pos++
	}

	// Проверить присутствие запрещенных ключевых слов
	if _, found := find(forbidden, string(field)); found {
		return nil, 0, errors.New("Forbidden keyword")
	}

	return statements, pos, nil
}

func parseOperator(query string, pos int, statements map[string][]string) (map[string][]string, int, error) {
	var (
		operator []byte = make([]byte, 0, 3)
	)

	for {
		if pos == len(query) {
			return nil, 0, errors.New("Expected EOL (;), got nothing")
		}

		if query[pos] == ' ' && len(operator) == 0 {
			pos++
			continue
		}

		if query[pos] == ' ' && len(operator) != 0 {
			statements["operators"] = append(statements["operators"], string(operator))

			pos++

			break
		}

		operator = append(operator, query[pos])
		pos++
	}

	// Проверить валидность оператора
	if _, found := find(operators, string(operator)); !found {
		return nil, 0, errors.New("Unknown operator")
	}

	return statements, pos, nil
}

func parseValue(query string, pos int, statements map[string][]string) (map[string][]string, int, error) {
	var (
		value []byte = make([]byte, 0, 15)

		err error
	)

LOOP:
	for {
		if pos == len(query) {
			return nil, 0, errors.New("Expected EOL (;), got nothing")
		}

		if query[pos] == ' ' && len(value) == 0 {
			pos++
			continue
		}

		switch query[pos] {
		case '\'', '"':
			quote := query[pos]
			pos++

			for query[pos] != quote {
				value = append(value, query[pos])
				pos++

				if pos == len(query) {
					return nil, 0, errors.New("Expected closing quote, got nothing")
				}
			}
			statements["values"] = append(statements["values"], string(value))

			pos++

			break LOOP

		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-', '.':
			if query[pos] == ' ' && len(value) == 0 {
				pos++
				continue
			}

			for query[pos] != ' ' && query[pos] != ';' {
				value = append(value, query[pos])
				pos++

				if pos == len(query) {
					return nil, 0, errors.New("Expected EOL (;), got nothing")
				}
			}

			var num float64

			if num, err = strconv.ParseFloat(string(value), 10); err != nil {
				return nil, 0, errors.New("Value is not number")
			}

			statements["values"] = append(statements["values"], fmt.Sprintf("%f", num))

			break LOOP

		default:
			return nil, 0, fmt.Errorf("Unknown character at position %d", pos)
		}
	}

	return statements, pos, nil
}

func parseSeparator(query string, pos int, statements map[string][]string) (map[string][]string, int, error) {
	var (
		separator []byte = make([]byte, 0, 3)
	)

	if pos == len(query) {
		return nil, 0, errors.New("Expected EOL (;), got nothing")
	}

	if query[pos] == ';' {
		return statements, pos, nil
	}

	for {
		if query[pos] == ' ' && len(separator) == 0 {
			pos++
			continue
		}

		if query[pos] == ' ' && len(separator) != 0 {
			statements["separators"] = append(statements["separators"], string(separator))

			pos++

			break
		}

		separator = append(separator, query[pos])
		pos++
	}

	// Проверить валидность сепаратора
	if _, found := find(separators, string(separator)); !found {
		return nil, 0, errors.New("Unknown separator")
	}

	return statements, pos, nil
}

// Функция парсинга выражений
func parseStatements(query string) (map[string][]string, error) {
	var (
		statements map[string][]string
		pos        int = 0

		err error
	)

	// Инициализация выражений
	statements = make(map[string][]string)
	statements["names"] = make([]string, 0, 3)
	statements["operators"] = make([]string, 0, 3)
	statements["values"] = make([]string, 0, 3)
	statements["separators"] = make([]string, 0, 2)

	for {
		if statements, pos, err = parseField(query, pos, statements); err != nil {
			return nil, err
		}

		if statements, pos, err = parseOperator(query, pos, statements); err != nil {
			return nil, err
		}

		if statements, pos, err = parseValue(query, pos, statements); err != nil {
			return nil, err
		}

		if statements, pos, err = parseSeparator(query, pos, statements); err != nil {
			return nil, err
		}

		// Конец условия WHERE
		if pos == len(query)-1 {
			break
		}
	}

	return statements, nil
}

// Parse - функция парсинга PostgreSQL-выражения части WHERE
// Принимает:
//	1. query - SQL запрос WHERE
//	2. qb - объект SelectBuilder
// Возвращает:
//	1. Указатель на qb
//	2. error/nil - флаг успеха или ошибку
func Parse(query string, qb squirrel.SelectBuilder) (*squirrel.SelectBuilder, error) {
	var (
		statements map[string][]string

		err error
	)

	// Распарсить условия в переданном WHERE
	if statements, err = parseStatements(query); err != nil {
		return nil, err
	}

	// Сформировать SELECT условие
	selectResult := squirrel.Select("*").From("some_table")

	var (
		sql    string
		wheres []string
	)

	// Сформировать WHERE условие на основе сепараторов
	if len(statements["separators"]) != 0 {
		for i, s := range statements["separators"] {
			where := fmt.Sprintf("%s %s %s %s",
				statements["names"][i], statements["operators"][i], statements["values"][i], s)
			wheres = append(wheres, where)
		}

		length := len(statements["names"])

		where := fmt.Sprintf("%s %s %s;",
			statements["names"][length-1], statements["operators"][length-1], statements["values"][length-1])

		wheres = append(wheres, where)
		sql, _, _ = selectResult.Where(strings.Join(wheres, " ")).ToSql()
	} else {
		where := fmt.Sprintf("%s %s %s;", statements["names"][0], statements["operators"][0], statements["values"][0])
		sql, _, _ = selectResult.Where(where).ToSql()
	}

	// Конечная строка SQL
	fmt.Println(sql)

	return &selectResult, nil
}
