package registry

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"vanishregistry-limpeza-cli/internal/config"
)

type CatalogResponse struct {
	Repositories []string `json:"repositories"`
}

type TagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func ParseTagDate(tag string) time.Time {
	// Itera sobre todos os padrões configurados no YAML
	for _, regra := range config.RegrasData {
		matches := regra.Regexp.FindStringSubmatch(tag)
		
		// Se achou o padrão, tenta extrair a data
		if len(matches) >= 2 {
			dataString := matches[1]
			
			// Usa o layout específico cadastrado para este regex no YAML
			t, err := time.Parse(regra.Layout, dataString)
			if err == nil {
				return t // Sucesso! Retorna a data e encerra o loop
			}
		}
	}

	// Se testou todos os padrões e nenhum funcionou (ex: "latest" ou fora do padrão), ignora a tag
	return time.Time{}
}

// -------------------------------------------------------------------
// FUNÇÃO AUXILIAR PARA INJETAR AUTENTICAÇÃO E IGNORAR SSL
// -------------------------------------------------------------------
func doRequest(req *http.Request) (*http.Response, error) {
	if config.Username != "" && config.Password != "" {
		req.SetBasicAuth(config.Username, config.Password)
	}

	// Cria um transporte customizado que ignora certificados expirados/inválidos
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	// Adiciona o transporte ao client
	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: tr, // Injeta a regra de ignorar o SSL aqui
	}
	
	return client.Do(req)
}

func FetchCatalog() ([]string, error) {
	req, err := http.NewRequest("GET", config.RegistryURL+"/v2/_catalog?n=1000", nil)
	if err != nil {
		return nil, err
	}

	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro HTTP %d ao buscar catálogo", resp.StatusCode)
	}

	var res CatalogResponse
	json.NewDecoder(resp.Body).Decode(&res)
	return res.Repositories, nil
}

func FetchTags(repo string) ([]string, error) {
	req, err := http.NewRequest("GET", config.RegistryURL+"/v2/"+repo+"/tags/list", nil)
	if err != nil {
		return nil, err
	}

	resp, err := doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res TagsResponse
	json.NewDecoder(resp.Body).Decode(&res)
	return res.Tags, nil
}

func GetDigest(repo, tag string) (string, error) {
	req, err := http.NewRequest("HEAD", config.RegistryURL+"/v2/"+repo+"/manifests/"+tag, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := doRequest(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return "", fmt.Errorf("Status 404")
	}
	return resp.Header.Get("Docker-Content-Digest"), nil
}

func DeleteByDigest(repo, digest string) error {
	req, err := http.NewRequest("DELETE", config.RegistryURL+"/v2/"+repo+"/manifests/"+digest, nil)
	if err != nil {
		return err
	}

	resp, err := doRequest(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 202 && resp.StatusCode != 200 {
		return fmt.Errorf("Erro HTTP %d", resp.StatusCode)
	}
	return nil
}