package shelltui

// sidebarItem representa um item de atalho na sidebar
type sidebarItem struct {
	key  string
	desc string
}

// sidebarSection representa uma seção da sidebar
type sidebarSection struct {
	title string
	items []sidebarItem
}

// sidebarSections retorna as seções de atalhos conforme a aba ativa
func sidebarSections(tab tabKind) []sidebarSection {
	global := sidebarSection{
		title: "Global",
		items: []sidebarItem{
			{key: "Tab", desc: "foco"},
			{key: "F1/Ctrl+H", desc: "ajuda"},
			{key: "Ctrl+B", desc: "sidebar"},
			{key: "Ctrl+L", desc: "limpar"},
			{key: "Ctrl+C", desc: "sair"},
		},
	}
	tabs := sidebarSection{
		title: "Abas",
		items: []sidebarItem{
			{key: "Ctrl+← / →", desc: "ciclar abas"},
			{key: "Ctrl+1", desc: "Shell"},
			{key: "Ctrl+2", desc: "Docker"},
			{key: "Ctrl+3", desc: "Kubernetes"},
			{key: "Ctrl+4", desc: "AWS"},
			{key: "Ctrl+5", desc: "Git"},
			{key: "Ctrl+6", desc: "Terraform"},
		},
	}

	switch tab {
	case tabDocker:
		return []sidebarSection{
			global, tabs,
			{
				title: "Docker",
				items: []sidebarItem{
					{key: "docker ps", desc: "containers"},
					{key: "docker ps -a", desc: "todos"},
					{key: "docker images", desc: "imagens"},
					{key: "docker stats", desc: "recursos"},
					{key: "docker logs", desc: "logs"},
					{key: "docker exec", desc: "entrar"},
					{key: "docker stop", desc: "parar"},
					{key: "docker rm", desc: "remover"},
					{key: "docker pull", desc: "baixar"},
					{key: "docker build", desc: "build"},
				},
			},
			{
				title: "Compose",
				items: []sidebarItem{
					{key: "dc up -d", desc: "subir"},
					{key: "dc down", desc: "parar"},
					{key: "dc logs -f", desc: "logs"},
					{key: "dc ps", desc: "status"},
				},
			},
		}

	case tabKubernetes:
		return []sidebarSection{
			global, tabs,
			{
				title: "Pods",
				items: []sidebarItem{
					{key: "k get pods", desc: "listar pods"},
					{key: "k get pods -A", desc: "todos ns"},
					{key: "k describe pod", desc: "detalhes"},
					{key: "k logs", desc: "logs pod"},
					{key: "k exec -it", desc: "entrar"},
					{key: "k delete pod", desc: "deletar"},
				},
			},
			{
				title: "Recursos",
				items: []sidebarItem{
					{key: "k get deploy", desc: "deployments"},
					{key: "k get svc", desc: "services"},
					{key: "k get ns", desc: "namespaces"},
					{key: "k get nodes", desc: "nós"},
					{key: "k apply -f", desc: "aplicar"},
					{key: "k rollout", desc: "rollout"},
				},
			},
		}

	case tabAWS:
		return []sidebarSection{
			global, tabs,
			{
				title: "AWS",
				items: []sidebarItem{
					{key: "aws whoami", desc: "identidade"},
					{key: "duck aws ec2", desc: "instâncias"},
					{key: "duck aws s3", desc: "buckets"},
					{key: "duck aws rds", desc: "databases"},
					{key: "duck aws ecs", desc: "containers"},
					{key: "duck aws logs", desc: "CloudWatch"},
				},
			},
		}

	case tabGit:
		return []sidebarSection{
			global, tabs,
			{
				title: "Git",
				items: []sidebarItem{
					{key: "git status", desc: "status"},
					{key: "git log", desc: "histórico"},
					{key: "git diff", desc: "diferenças"},
					{key: "git add .", desc: "stage all"},
					{key: "git commit", desc: "commit"},
					{key: "git push", desc: "push"},
					{key: "git pull", desc: "pull"},
					{key: "git branch", desc: "branches"},
					{key: "git checkout", desc: "trocar branch"},
					{key: "git stash", desc: "guardar"},
				},
			},
		}

	case tabTerraform:
		return []sidebarSection{
			global, tabs,
			{
				title: "Terraform",
				items: []sidebarItem{
					{key: "tf init", desc: "inicializar"},
					{key: "tf plan", desc: "planejar"},
					{key: "tf apply", desc: "aplicar"},
					{key: "tf destroy", desc: "destruir"},
					{key: "tf show", desc: "mostrar"},
					{key: "tf output", desc: "outputs"},
					{key: "tf state", desc: "estado"},
					{key: "tf fmt", desc: "formatar"},
					{key: "tf validate", desc: "validar"},
				},
			},
		}

	default: // tabShell
		return []sidebarSection{
			global, tabs,
			{
				title: "Duck",
				items: []sidebarItem{
					{key: "status", desc: "checar tools"},
					{key: "envcheck", desc: "ambiente"},
					{key: "netcheck", desc: "rede"},
					{key: "profile", desc: "perfis"},
					{key: "task", desc: "tarefas"},
					{key: "aliases", desc: "aliases"},
					{key: "history", desc: "histórico"},
					{key: "palette", desc: "buscar cmd"},
				},
			},
			{
				title: "Shell",
				items: []sidebarItem{
					{key: "pwd", desc: "diretório atual"},
					{key: "cd <dir>", desc: "mudar dir"},
					{key: "clear", desc: "limpar tela"},
					{key: "$ <cmd>", desc: "cmd nativo"},
				},
			},
		}
	}
}
