package gc



import (
	"os/exec"
	"strings"

	"vanishregistry-limpeza-cli/internal/config"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
)

func RunRouter() {
	if config.Environment == "k3s" {
		runK3s()
	} else {
		runDocker()
	}
}

func runDocker() {
	color.HiBlue("\n⚙️  Iniciando GC no Docker (Alvo: %s)...", config.TargetName)
	
	out, err := exec.Command("docker", "ps", "--filter", "name="+config.TargetName, "--format", "{{.ID}}").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		color.Red("❌ Container '%s' não encontrado: %v", config.TargetName, err)
		return
	}
	containerID := strings.Split(strings.TrimSpace(string(out)), "\n")[0]

	execCmd := func(args ...string) ([]byte, error) {
		cmdArgs := append([]string{"exec", containerID, "bin/registry", "garbage-collect"}, args...)
		cmdArgs = append(cmdArgs, "/etc/docker/registry/config.yml")
		return exec.Command("docker", cmdArgs...).CombinedOutput()
	}

	executeGC(execCmd)
}

func runK3s() {
	color.HiBlue("\n⚙️  Iniciando GC no K3s (Namespace: %s | Alvo: %s)...", config.K8sNamespace, config.TargetName)
	
	out, err := exec.Command("kubectl", "get", "pods", "-n", config.K8sNamespace, "-l", config.TargetName, "-o", "jsonpath={.items[0].metadata.name}").Output()
	if err != nil || strings.TrimSpace(string(out)) == "" {
		color.Red("❌ Pod não encontrado (namespace: %s, label: %s)", config.K8sNamespace, config.TargetName)
		return
	}
	podName := strings.TrimSpace(string(out))

	execCmd := func(args ...string) ([]byte, error) {
		cmdArgs := append([]string{"exec", "-n", config.K8sNamespace, podName, "--", "bin/registry", "garbage-collect"}, args...)
		cmdArgs = append(cmdArgs, "/etc/docker/registry/config.yml")
		return exec.Command("kubectl", cmdArgs...).CombinedOutput()
	}

	executeGC(execCmd)
}

func executeGC(runCmd func(...string) ([]byte, error)) {
	color.Yellow("🔍 Executando Dry-run...")
	if out, err := runCmd("--dry-run"); err != nil {
		color.Red("❌ Falha no Dry-run!\n%s", string(out))
		return
	}
	color.Green("✅ Dry-run OK.")

	if !config.IsAuto {
		prompt := promptui.Select{
			Label: "Deseja executar a limpeza REAL agora?",
			Items: []string{"Sim, executar agora", "Não, apenas sair"},
		}
		if _, confirm, _ := prompt.Run(); confirm != "Sim, executar agora" {
			color.Yellow("Operação abortada.")
			return
		}
	}

	color.HiMagenta("🚀 Executando GC real...")
	if out, err := runCmd(); err != nil {
		color.Red("❌ Erro no GC real!\n%s", string(out))
	} else {
		color.HiGreen("✨ DISCO LIMPO! Espaço recuperado com sucesso.")
	}
}