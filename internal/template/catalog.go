package template

import "strings"

type definition struct {
	ID          string
	Description string
	NextStep    string
	Files       map[string]string
}

func catalog() []definition {
	return []definition{
		dockerTemplate(),
		composeTemplate(),
		terraformTemplate(),
		jenkinsTemplate(),
		helmTemplate(),
		kubernetesTemplate(),
		goTemplate(),
	}
}

func findTemplate(id string) (definition, bool) {
	id = strings.ToLower(strings.TrimSpace(id))
	aliases := map[string]string{
		"dockerfile": "docker",
		"compose":    "compose",
		"docker-compose": "compose",
		"tf":         "terraform",
		"infra":      "terraform",
		"ci":         "jenkins",
		"pipeline":   "jenkins",
		"h":          "helm",
		"chart":      "helm",
		"k8s":        "kubernetes",
		"kube":       "kubernetes",
		"golang":     "go",
	}
	if mapped, ok := aliases[id]; ok {
		id = mapped
	}
	for _, item := range catalog() {
		if item.ID == id {
			return item, true
		}
	}
	return definition{}, false
}

func dockerTemplate() definition {
	def := goTemplate()
	def.ID = "docker"
	def.Description = "Dockerfile multi-stage para app Go"
	def.NextStep = "docker build -t {{ProjectName}} ."
	return def
}

func composeTemplate() definition {
	return definition{
		ID:          "compose",
		Description: "Docker Compose com app e banco PostgreSQL",
		NextStep:    "docker compose up -d --build",
		Files: map[string]string{
			"compose.yaml": `services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      APP_NAME: {{ProjectName}}
      DATABASE_URL: postgres://postgres:postgres@db:5432/{{ProjectNameLower}}?sslmode=disable
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: {{ProjectNameLower}}
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 10
    volumes:
      - db_data:/var/lib/postgresql/data

volumes:
  db_data:
`,
			"Dockerfile": dockerfileContent,
			".env.example": `APP_NAME={{ProjectName}}
DATABASE_URL=postgres://postgres:postgres@localhost:5432/{{ProjectNameLower}}?sslmode=disable
`,
			"README.md": `# {{ProjectName}}

Projeto criado com ` + "`duck template new compose`" + `.

` + "```sh" + `
docker compose up -d --build
docker compose ps
docker compose logs -f app
` + "```" + `
`,
		},
	}
}

func terraformTemplate() definition {
	return definition{
		ID:          "terraform",
		Description: "Terraform AWS basico (VPC + tags)",
		NextStep:    "terraform init && terraform plan",
		Files: map[string]string{
			"versions.tf": `terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}
`,
			"providers.tf": `provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = var.project_name
      ManagedBy   = "terraform"
      Environment = var.environment
    }
  }
}
`,
			"variables.tf": `variable "project_name" {
  type        = string
  description = "Nome do projeto"
  default     = "{{ProjectName}}"
}

variable "environment" {
  type        = string
  description = "Ambiente (dev, staging, prod)"
  default     = "dev"
}

variable "aws_region" {
  type        = string
  description = "Regiao AWS"
  default     = "us-east-1"
}

variable "vpc_cidr" {
  type        = string
  description = "CIDR da VPC"
  default     = "10.0.0.0/16"
}
`,
			"main.tf": `data "aws_availability_zones" "available" {
  state = "available"
}

resource "aws_vpc" "main" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.project_name}-vpc"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "${var.project_name}-igw"
  }
}

resource "aws_subnet" "public" {
  count                   = 2
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = data.aws_availability_zones.available.names[count.index]
  map_public_ip_on_launch = true

  tags = {
    Name = "${var.project_name}-public-${count.index + 1}"
  }
}
`,
			"outputs.tf": `output "vpc_id" {
  description = "ID da VPC"
  value       = aws_vpc.main.id
}

output "public_subnet_ids" {
  description = "Subnets publicas"
  value       = aws_subnet.public[*].id
}
`,
			"terraform.tfvars.example": `project_name = "{{ProjectName}}"
environment  = "dev"
aws_region   = "us-east-1"
vpc_cidr     = "10.0.0.0/16"
`,
			".gitignore": `.terraform/
*.tfstate
*.tfstate.*
.terraform.lock.hcl
terraform.tfvars
`,
			"README.md": `# {{ProjectName}} - Terraform

Projeto criado com ` + "`duck template new terraform`" + `.

` + "```sh" + `
cp terraform.tfvars.example terraform.tfvars
terraform init
terraform plan
terraform apply
` + "```" + `
`,
		},
	}
}

func jenkinsTemplate() definition {
	return definition{
		ID:          "jenkins",
		Description: "Jenkinsfile declarativo com Docker e testes",
		NextStep:    "commit Jenkinsfile e configure pipeline multibranch no Jenkins",
		Files: map[string]string{
			"Jenkinsfile": `pipeline {
  agent any

  environment {
    APP_NAME = '{{ProjectName}}'
    DOCKER_IMAGE = "{{ProjectNameLower}}"
  }

  stages {
    stage('Checkout') {
      steps {
        checkout scm
      }
    }

    stage('Test') {
      steps {
        sh 'echo "Adicione seus testes aqui"'
      }
    }

    stage('Build Docker') {
      when {
        expression { fileExists('Dockerfile') }
      }
      steps {
        sh 'docker build -t ${DOCKER_IMAGE}:${BUILD_NUMBER} .'
      }
    }

    stage('Deploy') {
      when {
        branch 'main'
      }
      steps {
        sh 'echo "Adicione deploy aqui (kubectl, compose, terraform, etc.)"'
      }
    }
  }

  post {
    always {
      cleanWs()
    }
    success {
      echo "Pipeline ${APP_NAME} concluido com sucesso"
    }
    failure {
      echo "Pipeline ${APP_NAME} falhou"
    }
  }
}
`,
			"docker-compose.jenkins.yaml": `services:
  jenkins:
    image: jenkins/jenkins:lts
    ports:
      - "8080:8080"
      - "50000:50000"
    volumes:
      - jenkins_home:/var/jenkins_home
      - /var/run/docker.sock:/var/run/docker.sock

volumes:
  jenkins_home:
`,
			"README.md": `# {{ProjectName}} - Jenkins

Projeto criado com ` + "`duck template new jenkins`" + `.

## Jenkins local (opcional)

` + "```sh" + `
docker compose -f docker-compose.jenkins.yaml up -d
` + "```" + `

Configure um pipeline multibranch apontando para este repositorio.
`,
		},
	}
}

func helmTemplate() definition {
	return definition{
		ID:          "helm",
		Description: "Chart Helm basico (Deployment + Service)",
		NextStep:    "helm lint ./chart && helm template {{ProjectName}} ./chart",
		Files: map[string]string{
			"chart/Chart.yaml": `apiVersion: v2
name: {{ProjectNameLower}}
description: Chart Helm para {{ProjectName}}
type: application
version: 0.1.0
appVersion: "1.0.0"
`,
			"chart/values.yaml": `replicaCount: 1

image:
  repository: {{ProjectNameLower}}
  tag: latest
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 8080

resources: {}
`,
			"chart/templates/deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "{{ProjectNameLower}}.fullname" . }}
  labels:
    app.kubernetes.io/name: {{ include "{{ProjectNameLower}}.name" . }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ include "{{ProjectNameLower}}.name" . }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ include "{{ProjectNameLower}}.name" . }}
    spec:
      containers:
        - name: app
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - containerPort: {{ .Values.service.port }}
`,
			"chart/templates/service.yaml": `apiVersion: v1
kind: Service
metadata:
  name: {{ include "{{ProjectNameLower}}.fullname" . }}
spec:
  type: {{ .Values.service.type }}
  selector:
    app.kubernetes.io/name: {{ include "{{ProjectNameLower}}.name" . }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.port }}
`,
			"chart/templates/_helpers.tpl": `{{- define "{{ProjectNameLower}}.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "{{ProjectNameLower}}.fullname" -}}
{{- printf "%s-%s" .Release.Name (include "{{ProjectNameLower}}.name" .) | trunc 63 | trimSuffix "-" -}}
{{- end -}}
`,
			"README.md": `# {{ProjectName}} - Helm

` + "```sh" + `
helm lint ./chart
helm template {{ProjectName}} ./chart
helm upgrade --install {{ProjectName}} ./chart
` + "```" + `
`,
		},
	}
}

func kubernetesTemplate() definition {
	return definition{
		ID:          "kubernetes",
		Description: "Manifestos Kubernetes + Kustomize",
		NextStep:    "kubectl apply -k .",
		Files: map[string]string{
			"kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
  - service.yaml

commonLabels:
  app.kubernetes.io/name: {{ProjectNameLower}}
  app.kubernetes.io/managed-by: duck
`,
			"deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ProjectNameLower}}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: {{ProjectNameLower}}
  template:
    metadata:
      labels:
        app: {{ProjectNameLower}}
    spec:
      containers:
        - name: app
          image: {{ProjectNameLower}}:latest
          ports:
            - containerPort: 8080
`,
			"service.yaml": `apiVersion: v1
kind: Service
metadata:
  name: {{ProjectNameLower}}
spec:
  selector:
    app: {{ProjectNameLower}}
  ports:
    - port: 80
      targetPort: 8080
  type: ClusterIP
`,
			"README.md": `# {{ProjectName}} - Kubernetes

` + "```sh" + `
kubectl apply -k .
kubectl get pods
` + "```" + `
`,
		},
	}
}

func goTemplate() definition {
	return definition{
		ID:          "go",
		Description: "API Go minima com healthcheck",
		NextStep:    "go run .",
		Files: map[string]string{
			"go.mod": `module {{ProjectNameLower}}

go 1.22
`,
			"main.go": goMainContent,
			"Dockerfile": dockerfileContent,
			".dockerignore": dockerignoreContent,
			"README.md": `# {{ProjectName}}

` + "```sh" + `
go run .
curl localhost:8080/health
docker build -t {{ProjectName}} .
` + "```" + `
`,
		},
	}
}

const goMainContent = `package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	name := getenv("APP_NAME", "{{ProjectName}}")
	addr := getenv("ADDR", ":8080")

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from %s\n", name)
	})

	log.Printf("%s listening on %s", name, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
`

const dockerignoreContent = `.git
.gitignore
.duck
.vscode
.idea
bin/
dist/
*.exe
`

const dockerfileContent = `# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 go build -o /out/app .

FROM alpine:3.20
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /out/app /app/app
USER appuser
EXPOSE 8080
ENV ADDR=:8080
CMD ["/app/app"]
`

