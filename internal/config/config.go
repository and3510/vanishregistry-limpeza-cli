package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Estrutura para os itens da lista de padrões
type PadraoData struct {
	Regex  string `yaml:"regex"`
	Layout string `yaml:"layout"`
}

type ArquivoRegras struct {
	Ambiente       string       `yaml:"ambiente"`
	RegistryURL    string       `yaml:"url_registry"`
	Alvo           string       `yaml:"alvo"`
	LimiteManter   int          `yaml:"limite_manter"`
	NamespaceK8s   string       `yaml:"namespace_k8s"`
	TagsProtegidas []string     `yaml:"tags_protegidas"`
	PadroesDeData  []PadraoData `yaml:"padroes_de_data"` // Agora é uma lista
	Username       string       `yaml:"usuario"`
	Password       string       `yaml:"senha"`
}

// Guarda o Regex compilado e seu respectivo Layout
type RegraProcessada struct {
	Regexp *regexp.Regexp
	Layout string
}

var (
	KeepCount     int
	K8sNamespace  string
	ProtectedTags = make(map[string]bool)
	RegrasData    []RegraProcessada // Lista carregada em memória
)

var (
	Environment string
	RegistryURL string
	TargetName  string
	IsAuto      bool
	Username    string
	Password    string
)

func CarregarRegras(caminho string) error {
	arquivo, err := os.ReadFile(caminho)
	if err != nil {
		return fmt.Errorf("não consegui ler o arquivo de regras: %v", err)
	}

	var regras ArquivoRegras
	if err := yaml.Unmarshal(arquivo, &regras); err != nil {
		return fmt.Errorf("YAML inválido: %v", err)
	}

	// Validações obrigatórias
	if regras.Ambiente == "" {
		return fmt.Errorf("campo 'ambiente' é obrigatório no YAML (valores: docker ou k3s)")
	}
	if regras.RegistryURL == "" {
		return fmt.Errorf("campo 'url_registry' é obrigatório no YAML")
	}
	if regras.Alvo == "" {
		return fmt.Errorf("campo 'alvo' é obrigatório no YAML")
	}
	if len(regras.PadroesDeData) == 0 {
		return fmt.Errorf("campo 'padroes_de_data' deve conter pelo menos um padrão")
	}

	if regras.Ambiente != "docker" && regras.Ambiente != "k3s" {
		return fmt.Errorf("'ambiente' inválido: '%s'. Use 'docker' ou 'k3s'", regras.Ambiente)
	}

	// Compila todos os padrões e guarda na memória
	for i, p := range regras.PadroesDeData {
		compiledRegex, err := regexp.Compile(p.Regex)
		if err != nil {
			return fmt.Errorf("erro ao compilar o regex do padrão %d ('%s'): %v", i+1, p.Regex, err)
		}
		RegrasData = append(RegrasData, RegraProcessada{
			Regexp: compiledRegex,
			Layout: p.Layout,
		})
	}

	// Popula variáveis globais
	Environment = regras.Ambiente
	RegistryURL = regras.RegistryURL
	TargetName = regras.Alvo
	KeepCount = regras.LimiteManter
	K8sNamespace = regras.NamespaceK8s
	Username = regras.Username
	Password = regras.Password

	for _, tag := range regras.TagsProtegidas {
		ProtectedTags[tag] = true
	}

	return nil
}