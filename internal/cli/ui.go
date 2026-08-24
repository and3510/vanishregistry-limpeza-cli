package cli

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"vanishregistry-limpeza-cli/internal/config"
	"vanishregistry-limpeza-cli/internal/gc"
	"vanishregistry-limpeza-cli/internal/registry"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

func Run() {
	if config.IsAuto {
		color.Cyan("🚀 Registry Cleaner CLI v3.1 — Modo Automático (%s)", strings.ToUpper(config.Environment))
		color.Cyan("🔗 URL: %s | 🎯 Alvo: %s\n", config.RegistryURL, config.TargetName)
		
		repos, err := registry.FetchCatalog()
		if err != nil {
			log.Fatalf("❌ Erro ao buscar catálogo: %v", err)
		}

		for _, repo := range repos {
			processRepository(repo)
		}
		gc.RunRouter()
		return
	}

	// Modo Interativo
	color.Cyan("🚀 Registry Cleaner CLI v3.1 — Modo Interativo (%s)", strings.ToUpper(config.Environment))
	color.Cyan("🔗 URL: %s | 🎯 Alvo: %s\n", config.RegistryURL, config.TargetName)

	repos, err := registry.FetchCatalog()
	if err != nil {
		log.Fatalf("❌ Erro fatal ao buscar catálogo: %v", err)
	}

	color.HiWhite("\n📊 ESTADO ATUAL DO REGISTRY:")
	printSummary(repos)

	modePrompt := promptui.Select{
		Label: "O que deseja fazer?",
		Items: []string{"Limpar TODOS os repositórios", "Limpar UMA imagem específica", "Sair"},
	}

	_, mode, err := modePrompt.Run()
	if err != nil || mode == "Sair" {
		color.Yellow("Operação abortada.")
		return
	}

	if mode == "Limpar TODOS os repositórios" {
		runCleanAll(repos)
	} else {
		runCleanSingle(repos)
	}

	gc.RunRouter()
	color.HiWhite("\nProcesso finalizado. Até logo!")
}

func runCleanAll(repos []string) {
	prompt := promptui.Select{
		Label: fmt.Sprintf("Confirma limpeza de TODOS os %d repositórios?", len(repos)),
		Items: []string{"Sim, limpar todos", "Não, voltar"},
	}
	if _, confirm, _ := prompt.Run(); confirm != "Sim, limpar todos" {
		color.Yellow("Operação abortada.")
		return
	}

	for _, repo := range repos {
		processRepository(repo)
	}
	color.HiCyan("\n✨ Varredura concluída!")
}

func runCleanSingle(repos []string) {
	var items []string
	for _, repo := range repos {
		tags, _ := registry.FetchTags(repo)
		items = append(items, fmt.Sprintf("%-50s [%d tags]", repo, len(tags)))
	}
	items = append(items, "← Cancelar")

	prompt := promptui.Select{
		Label:             "Selecione o repositório",
		Items:             items,
		Size:              15,
		StartInSearchMode: true,
		Searcher: func(input string, index int) bool {
			return strings.Contains(strings.ToLower(items[index]), strings.ToLower(input))
		},
	}

	idx, _, err := prompt.Run()
	if err != nil || idx == len(repos) {
		color.Yellow("Operação cancelada.")
		return
	}

	processRepository(repos[idx])
	color.HiCyan("\n✨ Limpeza concluída!")
}

func printSummary(repos []string) {
	fmt.Println(color.HiBlackString("---------------------------------------------------"))
	for _, repo := range repos {
		tags, _ := registry.FetchTags(repo)
		dateCount := 0
		for _, tag := range tags {
			if !config.ProtectedTags[tag] && !registry.ParseTagDate(tag).IsZero() {
				dateCount++
			}
		}

		msg := fmt.Sprintf("  • %-40s : %d tags (%d com data)", repo, len(tags), dateCount)
		if dateCount > config.KeepCount {
			color.Yellow(msg + " [SUJO]")
		} else {
			color.Green(msg + " [LIMPO]")
		}
	}
	fmt.Println(color.HiBlackString("---------------------------------------------------"))
}

func processRepository(repo string) {
	color.HiMagenta("\n📦 Limpando: %s", repo)
	allTags, err := registry.FetchTags(repo)
	if err != nil {
		color.Red("  ❌ Erro ao buscar tags: %v", err)
		return
	}

	var dateTags, keepTags []string
	for _, tag := range allTags {
		if config.ProtectedTags[tag] || registry.ParseTagDate(tag).IsZero() {
			keepTags = append(keepTags, tag)
		} else {
			dateTags = append(dateTags, tag)
		}
	}

	sort.Slice(dateTags, func(i, j int) bool {
		return registry.ParseTagDate(dateTags[i]).Before(registry.ParseTagDate(dateTags[j]))
	})

	if len(dateTags) <= config.KeepCount {
		color.Green("  ✔️ Nada a deletar (total de tags com data <= %d)", config.KeepCount)
		return
	}

	manter := dateTags[len(dateTags)-config.KeepCount:]
	deletar := dateTags[:len(dateTags)-config.KeepCount]

	protectedDigests := make(map[string]bool)
	for _, tag := range append(manter, keepTags...) {
		digest, _ := registry.GetDigest(repo, tag)
		if digest != "" {
			protectedDigests[digest] = true
		}
	}

	for _, tag := range deletar {
		digest, err := registry.GetDigest(repo, tag)
		if err != nil || protectedDigests[digest] {
			continue
		}
		if err = registry.DeleteByDigest(repo, digest); err == nil {
			color.Green("  ✔️ %s: eliminada", tag)
			protectedDigests[digest] = true
		} else {
			color.Red("  ❌ %s: Falha: %v", tag, err)
		}
	}
}