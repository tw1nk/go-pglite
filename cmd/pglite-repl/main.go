package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	_ "embed"

	"github.com/tw1nk/gopglite"
)

//go:embed testdata.sql
var tests string

func main() {
	pglite := gopglite.NewPGLite()

	defer pglite.Close()

	waitForStart, err := pglite.Start(gopglite.NewConfig())
	if err != nil {
		panic(err)
	}

	<-waitForStart

	// tests seems to be the only thing "working" right now.

	// run tests
	for _, line := range strings.Split(tests, "\n\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			fmt.Println("REPL:", line)
			out, err := pglite.Exec(line)
			if err != nil {
				panic(err)
			}
			fmt.Println("Response from pglite", string(out))
		}
	}

	fmt.Println("PGLite is ready")

	reader := bufio.NewReader(os.Stdin)
	for {
		input, err := reader.ReadString(';')
		if err != nil {
			panic(err)
		}

		if strings.TrimSpace(input) == "exit;" {
			goto shutdown
		}

		out, err := pglite.Exec(input)
		if err != nil {
			panic(err)
		}

		fmt.Println("Response from pglite", string(out))
	}

shutdown: // label
	err = pglite.Close()
	if err != nil {
		panic(err)
	}
}
