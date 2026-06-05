# Autocomplete do Duck

O jeito recomendado de habilitar autocomplete e usar `duck completion install`.

```sh
duck completion install powershell
duck completion install bash
duck completion install zsh
```

Esse comando adiciona o script de completion no perfil do shell do usuario:

- PowerShell: `Microsoft.PowerShell_profile.ps1`
- Bash: `~/.bashrc`
- Zsh: `~/.zshrc`

Depois de instalar, abra um novo terminal ou recarregue o perfil do shell.

Tambem e possivel apenas imprimir o script, sem instalar:

```sh
duck completion powershell
duck completion bash
duck completion zsh
```

Use `duck autocomplete` como alias de `duck completion` quando preferir:

```sh
duck autocomplete install powershell
```
