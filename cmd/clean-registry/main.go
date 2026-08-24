package main

import (
	"log"
	"os"

	"vanishregistry-limpeza-cli/internal/cli"
	"vanishregistry-limpeza-cli/internal/config"

	"github.com/fatih/color"
)

func main() {
	var configPath string

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--config":
			if i+1 < len(os.Args) {
				configPath = os.Args[i+1]
				i++
			}
		case "--auto":
			config.IsAuto = true
		}
	}

	if configPath == "" {
		color.Red("❌ Erro: parâmetro --config é obrigatório.")
		color.HiWhite("\nUso:")
		color.HiWhite("  ./limpeza-cli --config /etc/limpeza/regras.yaml")
		color.HiWhite("  ./limpeza-cli --config /etc/limpeza/regras.yaml --auto")
		os.Exit(1)
	}

	if err := config.CarregarRegras(configPath); err != nil {
		log.Fatalf("❌ Erro ao carregar configurações: %v", err)
	}

	cli.Run()
}
