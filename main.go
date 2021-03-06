package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/squirrel"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	var (
		qb squirrel.SelectBuilder

		sb *squirrel.SelectBuilder

		err error
	)

	for {
		// Считать выражение WHERE синтаксиса Postgres
		fmt.Print("> ")
		query, _ := reader.ReadString('\n')

		query = strings.Replace(query, "\n", "", -1)

		if strings.Compare("", query) == 0 {
			continue
		}

		// Условия завершения работы программы
		if strings.Compare("exit", strings.ToLower(query)) == 0 {
			break
		}

		// Распарсить выражение, вернуть *squirrel.SelectBuilder
		if sb, err = Parse(query, qb); err != nil {
			fmt.Println(err.Error())
			continue
		}

		fmt.Println(sb)
	}
}
