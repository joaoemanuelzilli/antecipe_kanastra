// Comando antecipe: CLI no estilo Unix (dados em stdout, logs em stderr).
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "uso: antecipe <gerar|exemplo|validar> [opções]")
	os.Exit(1)
}
